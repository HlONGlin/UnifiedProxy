package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/control"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func fleetTestApp(t *testing.T) *App {
	t.Helper()
	m := &CoreManager{dataDir: t.TempDir(), panelStarted: time.Now(), connectionStats: map[string]connectionSample{}, enforcedClients: map[string]bool{}}
	if err := m.load(); err != nil {
		t.Fatal(err)
	}
	m.remote = true
	a := &App{manager: m, role: "control", session: "fleet-test-session"}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := a.initializeFleet(ctx, m.dataDir); err != nil {
		t.Fatal(err)
	}
	return a
}
func fleetAuthRequest(method, path string, body []byte) *http.Request {
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "fleet-test-session"})
	return r
}

func TestFleetPanelAuthenticationAndManagedWriterLock(t *testing.T) {
	a := fleetTestApp(t)
	h := a.routes()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/fleet", nil))
	if w.Code != 401 {
		t.Fatal("unauthenticated fleet API exposed", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, fleetAuthRequest("GET", "/api/fleet", nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	n := &App{manager: a.manager, nodeRuntime: &nodeRuntime{manager: a.manager}, role: "node", session: "fleet-test-session"}
	if err := fleet.WriteJSON(filepath.Join(a.manager.dataDir, "agent.json"), map[string]string{"managed": "true"}); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	n.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/inbounds", []byte(`{}`)))
	if w.Code != 409 {
		t.Fatal("managed node writable", w.Code)
	}
	if err := fleet.WriteFile(filepath.Join(a.manager.dataDir, "break-glass"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	n.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/inbounds", []byte(`{}`)))
	if w.Code == 409 {
		t.Fatal("break-glass did not unlock writes")
	}
}

func TestFleetEditorRoutesInboundToolsToSelectedNode(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("remote-editor", "node.example:443")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/tools/inbound-port", ""},
		{http.MethodPost, "/api/tools/uuid", "{}"},
		{http.MethodPost, "/api/tools/reality-keypair", "{}"},
		{http.MethodPost, "/api/tools/vless-encryption", `{"authentication":"mlkem768"}`},
		{http.MethodPost, "/api/tools/subscription-path", "{}"},
	} {
		r := fleetAuthRequest(test.method, test.path, []byte(test.body))
		r.Header.Set("X-Fleet-Node", node.ID)
		w := httptest.NewRecorder()
		a.routes().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s returned %d: %s", test.method, test.path, w.Code, w.Body.String())
		}
	}
	if got := a.fleet.store.Node(node.ID).Desired; got != nil {
		t.Fatalf("editor helper unexpectedly published a node policy: %#v", got)
	}
}

func TestFleetEditorRoutesNodeScopedHelpers(t *testing.T) {
	a := fleetTestApp(t)
	for _, path := range []string{
		"/api/tools/uuid", "/api/tools/inbound-port", "/api/tools/subscription-path",
		"/api/tools/reality-keypair", "/api/tools/vless-encryption", "/api/tools/ech-keypair",
	} {
		if !a.remoteRoute(path) {
			t.Errorf("remoteRoute(%q) = false", path)
		}
	}
}

func TestFleetNativeRenderSubscriptionAndBackupRoundTrip(t *testing.T) {
	a := fleetTestApp(t)
	f := a.fleet
	for i, host := range []string{"one.example", "two.example", "three.example"} {
		n, err := f.store.AddNode("node-"+host, host)
		if err != nil {
			t.Fatal(err)
		}
		f.mu.Lock()
		ed, err := f.editor(n.ID)
		f.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		in := Inbound{ID: "in-" + host, Name: "socks", Type: "socks", Listen: "0.0.0.0", Port: 12000 + i, Enabled: true, Clients: []Client{{ID: "alice", Name: "alice", Username: "alice", Password: "original-password", Enabled: true}}}
		if err = ed.manager.saveInbound(in); err != nil {
			t.Fatal(err)
		}
		f.mu.Lock()
		desired, err := f.publish(n.ID, ed, nil)
		f.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		var model State
		if err = json.Unmarshal(desired.Model, &model); err != nil {
			t.Fatal(err)
		}
		yaml, err := renderStateConfig(model)
		if err != nil || string(yaml) != desired.MihomoYAML {
			t.Fatal("control/node render drift", err)
		}
	}
	if err := a.refreshFleetSubscriptions(); err != nil {
		t.Fatal(err)
	}
	a.manager.mu.Lock()
	token := a.manager.state.SubscriptionTokens["alice"]
	settings := a.manager.state.Settings
	a.manager.mu.Unlock()
	snapshot, _, ok := a.subscriptionSnapshot(subscriptionPublicPath(settings, token))
	if !ok || len(snapshot.Clients) != 3 {
		t.Fatal("fleet subscription missing nodes", len(snapshot.Clients))
	}
	proxies := subscriptionProxies(snapshot, "ignored.example")
	if len(proxies) != 3 {
		t.Fatal("missing proxies")
	}
	for _, p := range proxies {
		if p["server"] == "ignored.example" {
			t.Fatal("node address lost")
		}
	}
	backup, err := a.exportFleetBackup()
	if err != nil {
		t.Fatal(err)
	}
	if err = a.restoreFleetBackup(backup); err != nil {
		t.Fatal(err)
	}
	if len(f.store.Nodes()) != 3 {
		t.Fatal("backup lost nodes")
	}
	other, err := control.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if other.Identity.Public() == f.store.Identity.Public() {
		t.Fatal("identities collided")
	}
}

func TestFleetNativeApplyValidationRollbackAndAutonomy(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fleetcore.exe")
	cmd := exec.Command("go", "build", "-o", bin, "./internal/app/testdata/fleetcore")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, out)
	}
	dir := t.TempDir()
	m := &CoreManager{dataDir: dir, panelStarted: time.Now(), connectionStats: map[string]connectionSample{}, enforcedClients: map[string]bool{}}
	if err := m.load(); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	m.mu.Lock()
	m.state.Settings.CorePath = bin
	m.state.Settings.APIAddress = address
	m.state.Settings.APISecret = "local-only-secret"
	m.mu.Unlock()
	if err = m.writeConfig(); err != nil {
		t.Fatal(err)
	}
	if err = m.startCore(); err != nil {
		t.Fatal(err)
	}
	defer m.stopCore()
	n := &nodeRuntime{manager: m, sidecars: &agent.Sidecars{Dir: dir}}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = n.healthy(ctx); err != nil {
		t.Fatal(err)
	}
	makeDesired := func(name string, revision uint64) fleet.DesiredState {
		state := defaultState()
		state.Inbounds = []Inbound{{ID: "one", Name: name, Type: "socks", Listen: "127.0.0.1", Port: 18080, Enabled: true, Clients: []Client{{ID: "one", Name: "alice", Username: "alice", Password: "pass", Enabled: true, Total: 1000}}}}
		model, _ := json.Marshal(state)
		yaml, _ := renderStateConfig(state)
		d := fleet.DesiredState{NodeID: "test-node", Revision: revision, Model: model, MihomoYAML: string(yaml), Metering: "agent"}
		d.Digest, _ = d.ContentDigest()
		return d
	}
	good := makeDesired("good", 1)
	if err = n.Apply(ctx, good); err != nil {
		t.Fatal(err)
	}
	before, err := m.actualConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err = n.Apply(ctx, makeDesired("reject-me", 2)); err == nil {
		t.Fatal("bad config accepted")
	}
	after, _ := m.actualConfig()
	if !bytes.Equal(before, after) || n.applied.Revision != 1 {
		t.Fatal("failed validation changed live state")
	}
	if err = os.WriteFile(filepath.Join(dir, "fail-reload"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err = n.Apply(ctx, makeDesired("valid-but-reload-fails", 2)); err == nil {
		t.Fatal("reload failure accepted")
	}
	after, _ = m.actualConfig()
	if !bytes.Equal(before, after) || n.applied.Revision != 1 {
		t.Fatal("reload failure not rolled back")
	}
	if err = n.healthy(ctx); err != nil {
		t.Fatal("old service not healthy after rollback", err)
	}
	m.mu.Lock()
	m.state.Inbounds[0].Clients[0].Traffic.Down = 1001
	m.mu.Unlock()
	depleted, _ := m.renderConfig()
	if strings.Contains(string(depleted), `name: "good"`) {
		t.Fatal("quota enforcement depends on controller")
	}
	if n.manager.settingsSnapshot().APISecret != "local-only-secret" {
		t.Fatal("control overwrote local API credentials")
	}
	m.mu.Lock()
	expiry := time.Now().Add(72 * time.Hour).UnixMilli()
	m.state.Inbounds[0].Clients[0].ExpiryTime = expiry
	m.state.Inbounds = append(m.state.Inbounds, Inbound{ID: "local-only", Name: "local-only", Type: "socks", Listen: "127.0.0.1", Port: 18081, Enabled: true})
	m.mu.Unlock()
	if err = n.Apply(ctx, makeDesired("updated", 2)); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	current := copyState(m.state)
	m.mu.Unlock()
	if len(current.Inbounds) != 2 || current.Inbounds[0].Clients[0].ExpiryTime != expiry || current.Inbounds[0].Clients[0].Traffic.Down != 1001 {
		t.Fatal("apply lost unmanaged inbound or locally evolved expiry/counters")
	}
	candidate := bin + ".new"
	build := exec.Command("go", "build", "-ldflags=-X main.failStartup=1", "-o", candidate, "./internal/app/testdata/fleetcore")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failing candidate: %v %s", err, output)
	}
	oldBinary, _ := os.ReadFile(bin)
	if err := m.switchCoreBinary(ctx, candidate, bin); err == nil {
		t.Fatal("unhealthy kernel switch was acknowledged")
	}
	restoredBinary, _ := os.ReadFile(bin)
	if !bytes.Equal(oldBinary, restoredBinary) {
		t.Fatal("kernel failure did not restore previous executable")
	}
	if err := n.healthy(ctx); err != nil {
		t.Fatal("rollback did not restore healthy service", err)
	}
}

func TestFleetMeritImportPreservesCredentialsAndSidecarBackend(t *testing.T) {
	state, sidecars, _, err := convertMerit(meritNode{ID: "old", Ports: []meritPort{{ID: "mieru", Port: 12345, ProxyType: "mieru", Protocol: "TCP", Username: "original-user", Password: "original-password", Enabled: true}, {ID: "shadow", Port: 443, ProxyType: "shadowtls", Username: "user", Password: "tls-password", BackendAddr: "127.0.0.1:8388", BackendManaged: true, BackendPassword: "ss-password", BackendCipher: "aes-128-gcm", Enabled: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if state.Inbounds[0].Clients[0].Password != "original-password" || len(sidecars) != 2 || sidecars[0].Kind != "shadowsocks" {
		t.Fatal("legacy assets lost")
	}
}
