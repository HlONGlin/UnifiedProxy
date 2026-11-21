package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestNodeDNSProfilesMigratePreserveAndProtectReferences(t *testing.T) {
	dir := t.TempDir()
	legacy := fleetDNSCredentials{CloudflareToken: "legacy-token"}
	if err := fleet.WriteJSON(nodeDNSCredentialsPath(dir), legacy); err != nil {
		t.Fatal(err)
	}
	n := &nodeRuntime{manager: &CoreManager{dataDir: dir}}
	execute := func(payload string) (json.RawMessage, error) {
		return n.Execute(context.Background(), fleet.Task{Kind: "cert-credentials-manage", Encrypted: true, Payload: json.RawMessage(payload)})
	}
	if _, err := execute(`{"name":"account-b","cloudflareToken":"second-token"}`); err != nil {
		t.Fatal(err)
	}
	profiles, err := readNodeDNSProfiles(dir)
	if err != nil || len(profiles) != 2 || profiles["default"].CloudflareToken != "legacy-token" || profiles["account-b"].CloudflareToken != "second-token" {
		t.Fatalf("lost credentials during migration: %v", err)
	}
	views, _ := json.Marshal(n.dnsCredentialViews())
	if strings.Contains(string(views), "token") || !strings.Contains(string(views), "account-b") {
		t.Fatalf("credential status leaked secrets or lost names: %s", views)
	}
	if _, err = execute(`{"name":"account-b","cloudflareToken":"updated-token"}`); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(dir, "certificates", "edge", "acme")
	if err = os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatal(err)
	}
	cache := "ACCOUNT_EMAIL='acme@example.com'\nSAVED_CF_Token='updated-token'\nSAVED_CF_Account_ID='previous-account'\nCF_Zone_ID='previous-zone'\n"
	if err = os.WriteFile(filepath.Join(cacheDir, "account.conf"), []byte(cache), 0600); err != nil {
		t.Fatal(err)
	}
	certificates := map[string]nodeManagedCertificate{"edge": {Name: "edge", DNS: "dns_cf", CredentialName: "account-b", Automatic: true}}
	if err = saveNodeManagedCertificates(dir, certificates); err != nil {
		t.Fatal(err)
	}
	if _, err = execute(`{"name":"account-b","action":"delete"}`); err == nil {
		t.Fatal("deleted credentials used for certificate renewals")
	}
	if _, err = n.manageNodeCertificate("update", nodeCertificateManageRequest{Name: "edge", Automatic: false, CredentialName: "missing"}); err == nil {
		t.Fatal("accepted missing renewal credentials")
	}
	if _, err = n.manageNodeCertificate("update", nodeCertificateManageRequest{Name: "edge", Automatic: false, CredentialName: "default"}); err != nil {
		t.Fatal(err)
	}
	cleaned, err := os.ReadFile(filepath.Join(cacheDir, "account.conf"))
	if err != nil || strings.Contains(string(cleaned), "CF_") || !strings.Contains(string(cleaned), "ACCOUNT_EMAIL='acme@example.com'") {
		t.Fatalf("switching credentials retained old DNS cache or lost ACME settings: %s %v", cleaned, err)
	}
	certificates, err = readNodeManagedCertificates(dir)
	if err != nil || certificates["edge"].Automatic || certificates["edge"].CredentialName != "default" {
		t.Fatalf("renewal settings not saved: %v", err)
	}
	if _, err = execute(`{"name":"account-b","action":"delete"}`); err != nil {
		t.Fatal(err)
	}
	profiles, err = readNodeDNSProfiles(dir)
	if err != nil || len(profiles) != 1 || profiles["default"].CloudflareToken != "legacy-token" {
		t.Fatalf("delete affected other credentials: %v", err)
	}
	if _, err = execute(`{"name":"default","action":"delete"}`); err == nil {
		t.Fatal("deleted referenced legacy default credentials")
	}
	for _, payload := range []string{`{"name":"../bad","cloudflareToken":"x"}`, `{"name":"bad","cloudflareAccountId":"account-only"}`, `{"name":"bad","action":"unknown"}`} {
		if _, err = execute(payload); err == nil {
			t.Fatalf("accepted invalid payload %s", payload)
		}
	}
	if _, err = n.Execute(context.Background(), fleet.Task{Kind: "cert-credentials-manage", Payload: json.RawMessage(`{"name":"plain","cloudflareToken":"secret"}`)}); err == nil {
		t.Fatal("accepted unencrypted credentials")
	}
}

func TestNodeCertificateDeleteProtectsActualReferencesAndRemovesFiles(t *testing.T) {
	dir := t.TempDir()
	n := &nodeRuntime{manager: &CoreManager{dataDir: dir}}
	metadata := nodeManagedCertificate{Name: "edge", InboundIDs: []string{"stale-binding"}}
	if err := saveNodeManagedCertificates(dir, map[string]nodeManagedCertificate{"edge": metadata}); err != nil {
		t.Fatal(err)
	}
	certPath, keyPath, _ := nodeCertificatePaths(dir, "edge")
	if err := os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{certPath, keyPath} {
		if err := os.WriteFile(p, []byte("material"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	request := nodeCertificateManageRequest{Name: "edge"}
	n.manager.state.Inbounds = []Inbound{{ID: "direct-editor-binding", Certificate: certPath, PrivateKey: keyPath}}
	if _, err := n.manageNodeCertificate("delete", request); err == nil {
		t.Fatal("deleted a certificate bound via the inbound editor")
	}
	if got := n.managedCertificates(); len(got) != 1 || len(got[0].InboundIDs) != 1 || got[0].InboundIDs[0] != "direct-editor-binding" {
		t.Fatalf("status did not use actual bindings: %#v", got)
	}
	n.manager.state.Inbounds = nil
	model, _ := json.Marshal(State{Inbounds: []Inbound{{Certificate: nodeCertificateReference("edge", ".crt"), PrivateKey: nodeCertificateReference("edge", ".key")}}})
	n.applied = &fleet.DesiredState{Model: model}
	if _, err := n.manageNodeCertificate("delete", request); err == nil {
		t.Fatal("deleted a certificate referenced by applied state")
	}
	n.applied = nil
	if _, err := n.manageNodeCertificate("delete", request); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{certPath, keyPath} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("certificate material remains: %v", err)
		}
	}
	if got := n.managedCertificates(); len(got) != 0 {
		t.Fatal("deleted certificate remains managed")
	}
	if _, err := n.manageNodeCertificate("delete", request); err != nil {
		t.Fatalf("delete is not idempotent: %v", err)
	}
	if _, err := n.manageNodeCertificate("delete", nodeCertificateManageRequest{Name: "../escape"}); err == nil {
		t.Fatal("accepted path traversal")
	}
}

func TestFleetCertificateDeleteAndNodeManagementRoutes(t *testing.T) {
	a := fleetTestApp(t)
	certificate := testFleetCertificate(t)
	if err := a.fleet.saveCertificates(map[string]fleetCertificate{certificate.Name: certificate}); err != nil {
		t.Fatal(err)
	}
	call := func(path string, body any) *httptest.ResponseRecorder {
		data, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		a.routes().ServeHTTP(w, fleetAuthRequest(http.MethodPost, "/api/fleet/"+path, data))
		return w
	}
	if w := call("certificates/delete", map[string]string{"name": certificate.Name}); w.Code != 200 {
		t.Fatalf("control certificate delete: %d %s", w.Code, w.Body.String())
	}
	if certs, err := a.fleet.certificates(); err != nil || len(certs) != 0 {
		t.Fatal("control certificate still managed")
	}
	for _, action := range []string{"renew", "bind", "delete"} {
		if w := call("certificates/"+action, map[string]string{"name": "unknown"}); w.Code != 404 {
			t.Fatalf("unknown certificate %s: %d", action, w.Code)
		}
	}
	node, err := a.fleet.store.AddNode("edge", "edge.example")
	if err != nil {
		t.Fatal(err)
	}
	if err = a.fleet.store.UpdateStatus(node.ID, fleet.Status{Version: appVersion, Protocol: fleet.ProtocolVersion, Capabilities: []string{"cert-delete", "cert-update"}}); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"delete", "update"} {
		if w := call("nodes/"+node.ID+"/certificates/"+action, nodeCertificateManageRequest{Name: "edge"}); w.Code != 202 {
			t.Fatalf("node management %s: %d %s", action, w.Code, w.Body.String())
		}
	}
	editor, err := a.fleet.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	editor.manager.mu.Lock()
	editor.manager.state.Inbounds = []Inbound{{ID: "draft", Certificate: nodeCertificateReference("edge", ".crt"), PrivateKey: nodeCertificateReference("edge", ".key")}}
	editor.manager.mu.Unlock()
	if w := call("nodes/"+node.ID+"/certificates/delete", nodeCertificateManageRequest{Name: "edge"}); w.Code != 409 {
		t.Fatalf("draft binding not protected: %d %s", w.Code, w.Body.String())
	}
}

func TestNamedDNSCredentialTaskRemainsEncryptedThroughController(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("edge", "edge.example")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := fleet.NewEncryptionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err = a.fleet.store.UpdateStatus(node.ID, fleet.Status{Protocol: fleet.ProtocolVersion, Capabilities: []string{"cert-credentials-manage"}, EncryptionPublicKey: identity.Public()}); err != nil {
		t.Fatal(err)
	}
	taskID := fleet.ID()
	const kind = "cert-credentials-manage"
	plaintext := []byte(`{"name":"account-b","cloudflareToken":"secret-profile-token"}`)
	ciphertext, err := fleet.EncryptTaskPayload(identity.Public(), node.ID, taskID, kind, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"kind": kind, "taskId": taskID, "payload": json.RawMessage(ciphertext)})
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/fleet/nodes/"+node.ID+"/certificates/dns-credentials", body))
	if w.Code != 202 {
		t.Fatalf("credential management route: %d %s", w.Code, w.Body.String())
	}
	for _, task := range a.fleet.store.Tasks() {
		if task.Kind != kind || !task.Encrypted || len(task.Payload) != 0 {
			t.Fatal("task history exposed credential payload")
		}
	}
	backup, err := a.fleet.store.Backup()
	if err != nil || strings.Contains(string(backup), "secret-profile-token") {
		t.Fatal("controller persisted plaintext credentials")
	}
	if _, err = a.fleet.store.Enqueue(node.ID, kind, fleet.ID(), plaintext, time.Minute); err == nil {
		t.Fatal("unencrypted profile task accepted")
	}
}
