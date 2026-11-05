package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestFleetBackupPreservesUnpublishedDrafts(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("draft-node", "draft.example")
	if err != nil {
		t.Fatal(err)
	}
	ed, err := a.fleet.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	in := Inbound{ID: "draft", Name: "published", Type: "socks", Listen: "127.0.0.1", Port: 28080, Enabled: true}
	if err = ed.manager.saveInbound(in); err != nil {
		t.Fatal(err)
	}
	if _, err = a.fleet.publish(node.ID, ed, nil); err != nil {
		t.Fatal(err)
	}
	in.Name = "unpublished"
	if err = ed.manager.saveInbound(in); err != nil {
		t.Fatal(err)
	}
	a.fleet.editors = map[string]*App{}
	ed, err = a.fleet.editor(node.ID)
	if err != nil || ed.manager.state.Inbounds[0].Name != "unpublished" {
		t.Fatal("restart overwrote draft", err)
	}
	backup, err := a.exportFleetBackup()
	if err != nil {
		t.Fatal(err)
	}
	in.Name = "later"
	if err = ed.manager.saveInbound(in); err != nil {
		t.Fatal(err)
	}
	if err = a.restoreFleetBackup(backup); err != nil {
		t.Fatal(err)
	}
	ed, err = a.fleet.editor(node.ID)
	if err != nil || ed.manager.state.Inbounds[0].Name != "unpublished" {
		t.Fatal("restore lost draft", err)
	}
	var published State
	if err = json.Unmarshal(a.fleet.store.Node(node.ID).Desired.Model, &published); err != nil {
		t.Fatal(err)
	}
	if published.Inbounds[0].Name != "published" {
		t.Fatal("restore published a draft")
	}
}

func TestFleetInterruptedRestoreRecoversCorruptPartialDraft(t *testing.T) {
	a := fleetTestApp(t)
	n, err := a.fleet.store.AddNode("node", "node.example")
	if err != nil {
		t.Fatal(err)
	}
	ed, err := a.fleet.editor(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = ed.manager.saveInbound(Inbound{ID: "one", Name: "one", Type: "socks", Listen: "127.0.0.1", Port: 28080, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	backup, err := a.exportFleetBackup()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.fleet.dataDir, "fleet-nodes", n.ID, "state.json")
	if err = os.WriteFile(path, []byte("{partial"), 0600); err != nil {
		t.Fatal(err)
	}
	a.fleet.editors = map[string]*App{}
	if err = a.restoreFleetBackupTransaction(backup, false); err != nil {
		t.Fatal("startup recovery depended on broken current draft", err)
	}
	var restored State
	if err = fleet.ReadJSON(path, &restored); err != nil || len(restored.Inbounds) != 1 {
		t.Fatal("draft not restored", err)
	}
}

func TestFleetWebDAVUploadDownloadAndChecksumRejection(t *testing.T) {
	a := fleetTestApp(t)
	files := map[string][]byte{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "fixture" || password != "secret" {
			w.WriteHeader(401)
			return
		}
		switch r.Method {
		case "PUT":
			data, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(400)
				return
			}
			files[r.URL.Path] = data
			w.WriteHeader(201)
		case "GET":
			data, ok := files[r.URL.Path]
			if !ok {
				w.WriteHeader(404)
				return
			}
			_, _ = w.Write(data)
		default:
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	cfg := webDAVConfig{URL: server.URL + "/fleet.json", Username: "fixture", Password: "secret"}
	invoke := func(action string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"action": action, "webdav": cfg})
		w := httptest.NewRecorder()
		a.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/fleet/backup", body))
		return w
	}
	n, err := a.fleet.store.AddNode("original", "node.example")
	if err != nil {
		t.Fatal(err)
	}
	if w := invoke("upload"); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(files["/fleet.json"]) == 0 || len(files["/fleet.json.sha256"]) != 64 {
		t.Fatal("backup/checksum not uploaded")
	}
	if err = a.fleet.store.UpdateNode(n.ID, "changed", "node.example"); err != nil {
		t.Fatal(err)
	}
	if w := invoke("download"); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if a.fleet.store.Node(n.ID).Name != "original" {
		t.Fatal("WebDAV restore lost node")
	}
	files["/fleet.json.sha256"] = []byte("bad checksum")
	if w := invoke("download"); w.Code != 400 {
		t.Fatal("corrupt WebDAV accepted", w.Code)
	}
	if _, err := webDAVRequest(context.Background(), "GET", webDAVConfig{URL: cfg.URL, Username: "wrong"}, false, nil); err == nil {
		t.Fatal("unauthorized WebDAV accepted")
	}
}

func TestFleetMeritDuplicatePortAndIdentityRejected(t *testing.T) {
	for _, ports := range [][]meritPort{
		{{Port: 12345, Password: "one"}, {Port: 12345, Password: "two"}},
		{{ID: "same", Port: 12345, Password: "one"}, {ID: "same", Port: 23456, Password: "two"}},
		{{ID: "one", Port: 12345, Protocol: "tcp", Password: "one"}, {ID: "two", Port: 12345, Protocol: "TCP", Password: "two"}},
	} {
		if _, _, _, err := convertMerit(meritNode{ID: "legacy", Ports: ports}); err == nil {
			t.Fatal("ambiguous legacy ports accepted")
		}
	}
}

func TestFleetNaiveCertificateMustCoverActualDomain(t *testing.T) {
	a := fleetTestApp(t)
	c := testFleetCertificate(t)
	if err := a.fleet.saveCertificates(map[string]fleetCertificate{c.Name: c}); err != nil {
		t.Fatal(err)
	}
	n, err := a.fleet.store.AddNode("naive", "wrong.example")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.fleet.createSidecar(n.ID, fleetSidecarInput{Kind: "naive", Port: 443, Domain: "wrong.example", CertificateName: c.Name}); err == nil {
		t.Fatal("wrong domain certificate accepted")
	}
	if a.fleet.store.Node(n.ID).Desired != nil {
		t.Fatal("invalid recipe was published")
	}
}
