package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
	"mime/multipart"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLegacyRealApplySeedsSurviveRefreshResetAndZip(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "fixture-core")
	cmd := exec.Command("go", "build", "-o", binary, "./internal/app/testdata/fleetcore")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(output))
	}
	m := &CoreManager{dataDir: t.TempDir(), connectionStats: map[string]connectionSample{}, enforcedClients: map[string]bool{}}
	if err := m.load(); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	m.state.Settings.CorePath = binary
	m.state.Settings.APIAddress = address
	m.state.Settings.APISecret = "local-test"
	m.state.Settings.PanelPath = "/"
	defer m.stopCore()
	n := &nodeRuntime{manager: m, sidecars: &agent.Sidecars{Dir: m.dataDir}}
	a := &App{manager: m, standalone: &standaloneRuntime{runtime: n}, session: "fleet-test-session"}
	var b map[string]any
	_ = json.Unmarshal([]byte(oldestLegacyBackup), &b)
	port := b["nodes"].([]any)[0].(map[string]any)["ports"].([]any)[0].(map[string]any)
	port["trafficUsed"] = 123
	port["trafficLimit"] = 10000
	port["trafficResetAt"] = time.Now().UTC().Format(time.RFC3339)
	raw, _ := json.Marshal(b)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := fleetAuthRequest("POST", "/api/fleet/import/merit", raw)
		a.routes().ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		if m.state.Inbounds[0].Clients[0].Traffic.LegacyCombined != 123 || n.Snapshot().Traffic[m.state.Inbounds[0].ID+":"+m.state.Inbounds[0].Clients[0].ID].LegacyCombined != 123 {
			t.Fatal("real apply/status lost seed")
		}
	}
	w := httptest.NewRecorder()
	a.handleTrafficReset(w, httptest.NewRequest("POST", "/api/traffic/reset", strings.NewReader(`{"all":true}`)))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var desired fleet.DesiredState
	desired = *n.applied
	desired.Revision++
	desired.Digest, _ = desired.ContentDigest()
	if err = n.Apply(context.Background(), desired); err != nil {
		t.Fatal(err)
	}
	if m.state.Inbounds[0].Clients[0].Traffic.used() != 0 || m.state.Inbounds[0].Clients[0].Traffic.allTime() != 123 {
		t.Fatal("republish reset historical usage incorrectly")
	}
	zipResponse := httptest.NewRecorder()
	a.exportBackup(zipResponse)
	var upload bytes.Buffer
	form := multipart.NewWriter(&upload)
	part, _ := form.CreateFormFile("backup", "native.zip")
	_, _ = part.Write(zipResponse.Body.Bytes())
	form.Close()
	request := fleetAuthRequest("POST", "/api/backup", upload.Bytes())
	request.Header.Set("Content-Type", form.FormDataContentType())
	w = httptest.NewRecorder()
	a.routes().ServeHTTP(w, request)
	if w.Code != 200 {
		t.Fatal("native zip recovery", w.Code, w.Body.String())
	}
	if len(m.state.LegacyMigrations) != 1 || m.state.Inbounds[0].Clients[0].Traffic.allTime() != 123 {
		t.Fatal("zip round trip lost migration metadata/counters")
	}
}

// Shape taken from aa45551 (2026-10-05, first data-producing commit).
const oldestLegacyBackup = `{"admin":{"username":"old-admin","passwordSalt":"aa","passwordHash":"old-private-hash","createdAt":"2026-10-05T10:32:00+08:00"},"settings":{"subToken":"oldest-token-keep"},"nodes":[{"id":"oldest","name":"oldest node","apiKey":"old-private-api-key","address":"one.example","remark":"old notes","createdAt":"2026-10-05T10:32:00+08:00","ports":[{"id":"p1","port":31234,"protocol":"TCP","username":"old-alice","password":"original-password","label":"original listener","enabled":true,"createdAt":"2026-10-05T10:32:00+08:00"}]}]}`

func TestLegacyArchiveUnionsImportedNodes(t *testing.T) {
	var state State
	appendLegacyMigration(&state, []byte(oldestLegacyBackup), []string{"first"})
	stamp := state.LegacyMigrations[0].ImportedAt
	appendLegacyMigration(&state, []byte(oldestLegacyBackup), []string{"second", "first", "second"})
	if len(state.LegacyMigrations) != 1 || strings.Join(state.LegacyMigrations[0].NodeIDs, ",") != "first,second" || state.LegacyMigrations[0].ImportedAt != stamp {
		t.Fatalf("repeat import lost source associations: %+v", state.LegacyMigrations)
	}
}

func TestLegacyUnusableNaiveCannotBlockSupportedSiblings(t *testing.T) {
	a := fleetTestApp(t)
	var input map[string]any
	_ = json.Unmarshal([]byte(oldestLegacyBackup), &input)
	node := input["nodes"].([]any)[0].(map[string]any)
	node["ports"] = append(node["ports"].([]any), map[string]any{"id": "excluded-naive", "proxyType": "naive", "port": 0})
	raw, _ := json.Marshal(input)
	w := importLegacyTest(t, a, raw)
	if w.Code != 200 || len(a.fleet.store.Nodes()) != 1 || !strings.Contains(w.Body.String(), "excluded") || !bytes.Contains(a.manager.state.LegacyMigrations[0].Source, []byte("excluded-naive")) {
		t.Fatal("excluded Naive record blocked/lost supported backup data", w.Code, w.Body.String())
	}
}

func importLegacyTest(t *testing.T, a *App, raw []byte) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/fleet/import/merit", raw))
	return w
}
func TestLegacyOldestOperationalSubscriptionRestartAndRebackup(t *testing.T) {
	a := fleetTestApp(t)
	identity := a.manager.state.Settings
	w := importLegacyTest(t, a, []byte(oldestLegacyBackup))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	n := a.fleet.store.Nodes()[0]
	var state State
	_ = json.Unmarshal(n.Desired.Model, &state)
	if len(state.Inbounds) != 1 || state.Inbounds[0].Type != "mieru" || state.Inbounds[0].Clients[0].Password != "original-password" {
		t.Fatal("oldest credentials/type lost")
	}
	if a.manager.state.Settings.Username != identity.Username || a.manager.state.Settings.Password != identity.Password {
		t.Fatal("restore replaced operator identity")
	}
	for _, check := range []struct {
		path string
		code int
	}{{"/sub?token=oldest-token-keep", 200}, {"/sub?token=wrong", 403}} {
		w = httptest.NewRecorder()
		a.routes().ServeHTTP(w, httptest.NewRequest("GET", check.path, nil))
		if w.Code != check.code {
			t.Fatal(check.path, w.Code, w.Body.String())
		}
		if check.code == 200 && (!strings.Contains(w.Body.String(), "original-password") || !strings.Contains(w.Body.String(), "one.example")) {
			t.Fatal("old subscription does not contain original endpoint")
		}
	}
	w = httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest("GET", "/api/state", nil))
	if strings.Contains(w.Body.String(), "old-private") || strings.Contains(string(n.Desired.Model), "old-private") {
		t.Fatal("recovery archive leaked")
	}
	backup, err := a.exportFleetBackup()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(backup, []byte("old-private-api-key")) {
		t.Fatal("original recovery source lost")
	}
	a.manager.state.LegacyMigrations = nil
	if err = a.restoreFleetBackup(backup); err != nil {
		t.Fatal(err)
	}
	m := &CoreManager{dataDir: a.manager.dataDir}
	if err = m.load(); err != nil {
		t.Fatal(err)
	}
	a.manager = m
	w = httptest.NewRecorder()
	a.routes().ServeHTTP(w, httptest.NewRequest("GET", "/sub?token=oldest-token-keep", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "original-password") {
		t.Fatal("restart/rebackup broke subscription", w.Code, w.Body.String())
	}
}

func TestLegacyTrafficSeedQuotaAndResetAreIdempotent(t *testing.T) {
	node := meritNode{ID: "traffic", Ports: []meritPort{{ID: "p", Port: 31234, Username: "alice", Password: "pass", Enabled: false, AutoDisabled: true, TrafficUsed: 100, TrafficLimit: 100, TrafficResetAt: "2026-11-01T00:00:00Z"}}}
	converted, _, _, err := convertMerit(node)
	if err != nil {
		t.Fatal(err)
	}
	in := converted.Inbounds[0]
	if !in.Enabled || !clientDepleted(in.Clients[0], time.Now().UnixMilli()) || in.Clients[0].Traffic.Up != 0 || in.Clients[0].Traffic.Down != 0 {
		t.Fatal("quota lost or directional bytes fabricated")
	}
	desired := copyState(converted)
	desired.Inbounds[0].Traffic = TrafficStats{}
	desired.Inbounds[0].Clients[0].Traffic = TrafficStats{}
	seedLegacyTraffic(&desired, State{})
	if desired.Inbounds[0].Clients[0].Traffic.used() != 100 {
		t.Fatal("first apply failed to seed")
	}
	old := copyState(desired)
	old.Inbounds[0].Clients[0].Traffic.LegacyCombined = 0
	old.Inbounds[0].Traffic.LegacyCombined = 0
	next := copyState(converted)
	next.Inbounds[0].Traffic = TrafficStats{}
	next.Inbounds[0].Clients[0].Traffic = TrafficStats{}
	seedLegacyTraffic(&next, old)
	if next.Inbounds[0].Clients[0].Traffic.used() != 0 {
		t.Fatal("republish resurrected cleared historical usage")
	}
	m := &CoreManager{dataDir: t.TempDir(), state: converted}
	m.state.Inbounds[0].LastTrafficReset = "2000-01-01T00:00:00Z"
	m.applyScheduledTrafficResets()
	if m.state.Inbounds[0].Clients[0].Traffic.used() != 0 || m.state.Inbounds[0].Clients[0].Traffic.allTime() != 100 {
		t.Fatal("monthly reset lost all-time bytes or kept quota depleted")
	}
}

func TestLegacyInvalidBatchAndPublishFailureRollBackEverything(t *testing.T) {
	a := fleetTestApp(t)
	var b map[string]any
	_ = json.Unmarshal([]byte(oldestLegacyBackup), &b)
	b["nodes"] = append(b["nodes"].([]any), map[string]any{"id": "bad", "ports": []any{map[string]any{"port": 0}}})
	raw, _ := json.Marshal(b)
	if w := importLegacyTest(t, a, raw); w.Code != 400 || len(a.fleet.store.Nodes()) != 0 {
		t.Fatal("invalid batch partially imported", w.Code, w.Body.String())
	}
	// First node passes, second fails certificate loading during publish.
	cert := testFleetCertificate(t)
	_ = json.Unmarshal([]byte(oldestLegacyBackup), &b)
	b["nodes"] = append(b["nodes"].([]any), map[string]any{"id": "cert-failure", "name": "bad-cert", "address": "two.example", "ports": []any{map[string]any{"id": "p2", "port": 31235, "proxyType": "anytls", "username": "b", "password": "b", "enabled": true, "tlsCert": cert.Certificate, "tlsKey": "PRIVATE KEY missing-file"}}})
	raw, _ = json.Marshal(b)
	w := importLegacyTest(t, a, raw)
	if w.Code != 500 || len(a.fleet.store.Nodes()) != 0 || len(a.manager.state.LegacyMigrations) != 0 {
		t.Fatal("failed publish left partial import", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(a.manager.dataDir, "fleet-restore-transaction.json")); !os.IsNotExist(err) {
		t.Fatal("completed rollback retained journal", err)
	}
}

func TestLegacyLaterEgressTLSAndUnsupportedSibling(t *testing.T) {
	cert := testFleetCertificate(t)
	n := meritNode{ID: "mixed", Ports: []meritPort{
		{ID: "m", Port: 31234, Username: "alice", Password: "pass", Enabled: true},
		{ID: "a", Port: 31235, ProxyType: "anytls", Username: "bob", Password: "tls-pass", Enabled: true, TLSCert: cert.Certificate, TLSKey: cert.PrivateKey, Domain: "old.example", TLSFingerprint: "firefox"},
		{ID: "n", Port: 31236, ProxyType: "naive", Password: "archived"},
		{ID: "s", Port: 31237, ProxyType: "shadowtls", Password: "tls", BackendAddr: "127.0.0.1:31238", BackendPassword: "ss", BackendManaged: true, Enabled: true},
	}, Egress: &meritEgress{Proxies: []meritEgressProxy{{ID: "eg", Name: "landing", Host: "127.0.0.1", Port: 1080, Username: "socks-user", Password: "socks-pass", Enabled: true}}, Rules: []meritEgressRule{{ID: "r", Domains: []string{"example.org"}, Action: "PROXY", ProxyNames: []string{"landing"}, Ports: []int{31234}, Enabled: true}}}}
	state, sidecars, logs, err := convertMerit(n)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Inbounds) != 2 || len(sidecars) != 2 || sidecars[1].Cipher != "aes-256-gcm" || !strings.Contains(strings.Join(logs, " "), "excluded") {
		t.Fatal("supported siblings lost", logs)
	}
	if len(state.Outbounds) != 1 || state.Outbounds[0].Password != "socks-pass" || len(state.RoutingRules) != 2 || !strings.Contains(state.RoutingRules[0].Payload, "DOMAIN-SUFFIX,example.org") || !strings.Contains(state.RoutingRules[0].Payload, "IN-NAME,"+state.Inbounds[0].Name) {
		t.Fatal("egress order/scope/credentials lost", state.RoutingRules)
	}
	snapshot := subscriptionSnapshot{Clients: []subscriptionClient{{Inbound: state.Inbounds[1], Client: state.Inbounds[1].Clients[0], Server: "one.example"}}}
	yaml, err := buildMihomoSubscription(snapshot, "one.example")
	if err != nil || !bytes.Contains(yaml, []byte("firefox")) || !bytes.Contains(yaml, []byte("old.example")) {
		t.Fatal("AnyTLS SNI/fingerprint lost", err, string(yaml))
	}
}

func TestLegacyJSONBackupUploadAndMalformedInput(t *testing.T) {
	a := fleetTestApp(t)
	for _, test := range []struct {
		raw  string
		code int
	}{{"{bad", 400}, {"{\"nodes\":null}", 400}, {"{\"nodes\":[]} trailing", 400}, {oldestLegacyBackup, 200}} {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, _ := form.CreateFormFile("backup", "oldest.json")
		_, _ = part.Write([]byte(test.raw))
		form.Close()
		request := fleetAuthRequest("POST", "/api/backup", body.Bytes())
		request.Header.Set("Content-Type", form.FormDataContentType())
		w := httptest.NewRecorder()
		a.routes().ServeHTTP(w, request)
		if w.Code != test.code {
			t.Fatal(w.Code, test.code, w.Body.String())
		}
	}
	if len(a.fleet.store.Nodes()) != 1 {
		t.Fatal("normal backup uploader did not import legacy store")
	}
}
