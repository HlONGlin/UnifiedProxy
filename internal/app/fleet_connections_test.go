package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestFleetDisabledSessionsCloseOnlyAffectedInboundAndUser(t *testing.T) {
	before := State{Inbounds: []Inbound{
		{ID: "one", Name: "One", Enabled: true, Clients: []Client{{ID: "a", Name: "Alice", Username: "alice", Enabled: true}, {ID: "b", Name: "Bob", Username: "bob", Enabled: true}}},
		{ID: "two", Name: "Two", Enabled: true},
		{ID: "keep", Name: "Keep", Enabled: true},
	}}
	after := copyState(before)
	after.Inbounds[0].Clients[0].Enabled = false
	after.Inbounds[1].Enabled = false
	var deleted []string
	var deletedMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer local-secret" {
			t.Error("core request lost local authentication")
		}
		if r.Method == "GET" && r.URL.Path == "/connections" {
			_ = json.NewEncoder(w).Encode(map[string]any{"connections": []any{
				map[string]any{"id": "alice-session", "metadata": map[string]string{"inboundName": "One", "inboundUser": "alice"}},
				map[string]any{"id": "bob-session", "metadata": map[string]string{"inboundName": "One", "inboundUser": "bob"}},
				map[string]any{"id": "two-session", "metadata": map[string]string{"inboundName": "Two"}},
				map[string]any{"id": "keep-session", "metadata": map[string]string{"inboundName": "Keep"}},
				map[string]any{"id": "unknown-session", "metadata": map[string]string{}},
			}})
			return
		}
		if r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/connections/") {
			deletedMu.Lock()
			deleted = append(deleted, strings.TrimPrefix(r.URL.Path, "/connections/"))
			deletedMu.Unlock()
			w.WriteHeader(204)
			return
		}
		t.Error("unexpected core request", r.Method, r.URL)
		w.WriteHeader(400)
	}))
	defer server.Close()
	m := &CoreManager{state: State{Settings: Settings{APIAddress: strings.TrimPrefix(server.URL, "http://"), APISecret: "local-secret"}}}
	n := &nodeRuntime{manager: m}
	if err := n.closeRestrictedSessions(context.Background(), before, after); err != nil {
		t.Fatal(err)
	}
	deletedMu.Lock()
	defer deletedMu.Unlock()
	if !reflect.DeepEqual(deleted, []string{"alice-session", "two-session"}) {
		t.Fatal("unrelated sessions closed or disabled sessions retained", deleted)
	}
}

func TestFleetInboundSwitchReturnsAppliedTaskReceipt(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("Switch", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{true, false} {
		input, _ := json.Marshal(Inbound{ID: "entry", Name: "Entry", Type: "socks", Listen: "127.0.0.1", Port: 12345, Enabled: enabled})
		r := httptest.NewRequest("POST", "/api/inbounds", strings.NewReader(string(input)))
		w := httptest.NewRecorder()
		a.handleRemotePanel(w, r, node.ID)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var response struct {
			Data struct {
				Task struct {
					ID, NodeID, Kind, Status string
				} `json:"fleetTask"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		task := response.Data.Task
		if task.ID == "" || task.NodeID != node.ID || task.Kind != "apply-state" || task.Status != "pending" {
			t.Fatal("switch must return the actual queued task", w.Body.String())
		}
		var model State
		_ = json.Unmarshal(a.fleet.store.Node(node.ID).Desired.Model, &model)
		if model.Inbounds[0].Enabled != enabled {
			t.Fatal("enabled flag was lost before publishing", enabled)
		}
	}
}
