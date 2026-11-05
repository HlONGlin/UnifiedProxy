package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestFleetWorkspaceBackupRestoresCredentialFreeNodePolicy(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("backup", "backup.example")
	if err != nil {
		t.Fatal(err)
	}
	a.fleet.mu.Lock()
	editor, err := a.fleet.editor(node.ID)
	a.fleet.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	editor.manager.mu.Lock()
	editor.manager.state.Settings.Username = ""
	editor.manager.state.Settings.Password = ""
	editor.manager.mu.Unlock()
	if err := editor.manager.saveInbound(Inbound{ID: "backup-entry", Name: "Node backup", Type: "socks", Listen: "127.0.0.1", Port: 12345, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	editor.exportBackup(w)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	file, err := os.CreateTemp(t.TempDir(), "node-backup-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err = file.Write(w.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err = file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = editor.importBackup(file, false); err != nil {
		t.Fatal("node backup rejected", err)
	}
	if len(editor.manager.state.Inbounds) != 1 || editor.manager.state.Inbounds[0].ID != "backup-entry" {
		t.Fatal("node policy not restored")
	}
	if len(a.manager.state.Inbounds) != 0 {
		t.Fatal("node backup changed the host")
	}
	if _, err = file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = a.importBackup(file, false); err == nil {
		t.Fatal("host accepted a backup without its required login credentials")
	}
}

func TestFleetWorkspaceRoutesBindServerAndShareOnlyLogin(t *testing.T) {
	a := fleetTestApp(t)
	a.manager.state.Settings.PanelPath = "/console/"
	one, _ := a.fleet.store.AddNode("one", "one.example")
	two, _ := a.fleet.store.AddNode("two", "two.example")
	for i, id := range []string{one.ID, two.ID} {
		a.fleet.mu.Lock()
		ed, err := a.fleet.editor(id)
		a.fleet.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		if err := ed.manager.saveInbound(Inbound{ID: "entry", Name: []string{"One inbound", "Two inbound"}[i], Type: "socks", Listen: "0.0.0.0", Port: 12000 + i, Enabled: true}); err != nil {
			t.Fatal(err)
		}
		details, _ := json.Marshal(map[string]any{"system": map[string]any{"cpuPercent": 11 + i*22}, "logs": []string{[]string{"one-log", "two-log"}[i]}, "cpuHistory": map[string]any{"30": []cpuSample{{T: time.Now().Unix(), Cpu: float64(i + 1)}}}})
		if err := a.fleet.store.UpdateStatus(id, fleet.Status{Running: true, Version: "test", Details: details, Traffic: map[string]fleet.Usage{"entry": {Up: uint64(100 + i), Down: 200, AllTimeUp: 300, AllTimeDown: 400}}}); err != nil {
			t.Fatal(err)
		}
	}
	h := a.routes()
	request := func(path string, auth bool, header string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if auth {
			r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "fleet-test-session"})
		}
		if header != "" {
			r.Header.Set("X-Fleet-Node", header)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("/console/", true, ""); w.Code != 200 || !strings.Contains(w.Body.String(), `data-workspace="portal"`) || strings.Contains(w.Body.String(), `id="view-dashboard"`) {
		t.Fatal("control root is not a separate fleet portal", w.Code)
	}
	for _, id := range []string{one.ID, two.ID} {
		path := "/console/nodes/" + id + "/"
		if w := request(path, false, ""); w.Code != 302 || !strings.HasPrefix(w.Header().Get("Location"), "/console/login?next=") {
			t.Fatal("node must share the control login", w.Code, w.Header())
		}
		if w := request(path, true, ""); w.Code != 200 || !strings.Contains(w.Body.String(), `data-node-id="`+id+`"`) {
			t.Fatal("authenticated card unexpectedly requires another login", w.Code)
		}
	}
	path := "/console/nodes/" + one.ID + "/"
	if w := request(path+"api/state", false, ""); w.Code != 401 {
		t.Fatal("node state exposed without login", w.Code)
	}
	w := request(path+"api/state", true, two.ID)
	var state struct {
		Data struct {
			FleetNode string `json:"fleetNode"`
			NodeName  string `json:"nodeName"`
			State     State  `json:"state"`
			System    struct {
				CPU float64 `json:"cpuPercent"`
			} `json:"system"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || state.Data.FleetNode != one.ID || state.Data.NodeName != "one" || state.Data.System.CPU != 11 || state.Data.State.Inbounds[0].Name != "One inbound" || state.Data.State.Traffic.Up != 100 || state.Data.State.Inbounds[0].Traffic.Down != 200 {
		t.Fatal("server URL did not isolate config and telemetry", w.Body.String())
	}
	for _, tc := range []struct{ path, want string }{{"api/core/logs", "one-log"}, {"api/logs/download", "one-log"}, {"api/server/cpuHistory/30", `"cpu":1`}, {"api/config/download", "12000"}} {
		w := request(path+tc.path, true, two.ID)
		if w.Code != 200 || !strings.Contains(w.Body.String(), tc.want) || strings.Contains(w.Body.String(), "two-log") {
			t.Fatal(tc.path, w.Code, w.Body.String())
		}
	}
	for _, endpoint := range []string{"api/auth/credentials", "api/panel/restart", "api/fleet/backup", "api/client-mesh", "api/server/cpuHistory/999"} {
		w := request(path+endpoint, true, "")
		if w.Code != 404 && w.Code != 400 {
			t.Fatal("node workspace fell through to a control resource", endpoint, w.Code)
		}
	}
	if w := request(path+"api/config", true, ""); w.Code != 200 || strings.Contains(w.Body.String(), "fleet-test-session") {
		t.Fatal("node settings unavailable", w.Code)
	}
	if w := request("/console/host/", true, two.ID); w.Code != 200 || !strings.Contains(w.Body.String(), `data-workspace="host"`) {
		t.Fatal("host settings inaccessible", w.Code)
	}
	if w := request("/nodes/"+one.ID+"/", true, ""); w.Code != 404 {
		t.Fatal("custom panel prefix bypass", w.Code)
	}
	if w := request("/console/nodes/missing/", true, ""); w.Code != 404 {
		t.Fatal("unknown server rendered a workspace", w.Code)
	}
	if err := a.fleet.store.Revoke(one.ID); err != nil {
		t.Fatal(err)
	}
	if w := request(path, true, ""); w.Code != 404 {
		t.Fatal("revoked server still accessible", w.Code)
	}
}

func TestFleetWorkspaceLoginDestinationIsLocalAndValidated(t *testing.T) {
	a := fleetTestApp(t)
	a.manager.state.Settings.PanelPath = "/console/"
	for _, next := range []string{"https://evil.example/", "//evil.example/", "../host/", "nodes/../../", `nodes/a\b/`, "nodes/a/api/state", "host/api/backup"} {
		if got := a.workspaceDestination(next); got != "/console/" {
			t.Fatal("unsafe login redirect", next, got)
		}
	}
	for _, next := range []string{"host/", "nodes/server-01/"} {
		if got := a.workspaceDestination(next); got != "/console/"+next {
			t.Fatal("workspace lost after login", next, got)
		}
	}
}
