package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Opt-in live browser fixture. Uses real authenticated handlers/persistence.
// Only the WARP provider and the core executable are local test doubles; this
// intentionally does not certify external CA/DNS/VPN/provider availability.
func TestDesignAcceptanceServer(t *testing.T) {
	address := os.Getenv("UNIFIEDPROXY_DESIGN_ADDRESS")
	if address == "" {
		t.Skip("set UNIFIEDPROXY_DESIGN_ADDRESS for opt-in browser acceptance")
	}
	dir := os.Getenv("UNIFIEDPROXY_DESIGN_DATA")
	if dir == "" {
		t.Fatal("UNIFIEDPROXY_DESIGN_DATA required")
	}
	m := &CoreManager{dataDir: dir, panelStarted: time.Now(), connectionStats: map[string]connectionSample{}, enforcedClients: map[string]bool{}}
	if err := m.load(); err != nil {
		t.Fatal(err)
	}
	m.state.Settings.PanelListen = address
	m.state.Settings.PanelPath = "/"
	m.state.Settings.CorePath = os.Getenv("UNIFIEDPROXY_DESIGN_CORE")
	m.state.Settings.APIAddress = os.Getenv("UNIFIEDPROXY_DESIGN_CORE_API")
	if m.state.Settings.CorePath == "" || m.state.Settings.APIAddress == "" {
		t.Fatal("provide Linux fixture core and API address")
	}
	if err := m.saveLocked(); err != nil {
		t.Fatal(err)
	}
	if err := m.writeConfig(); err != nil {
		t.Fatal(err)
	}
	if err := m.startCore(); err != nil {
		t.Fatal(err)
	}
	defer m.stopCore()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "PUT" {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(WarpAccountInfo{License: body["license"], AccountType: "unlimited"})
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		_ = json.NewEncoder(w).Encode(warpFixture())
	}))
	defer provider.Close()
	m.warpClient = &warpClient{baseURL: provider.URL, httpClient: provider.Client()}
	a := &App{manager: m, session: "design-fixture-session"}
	var err error
	a.peers, err = newPeerNetwork(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.clients, err = newClientMesh(dir, a.peers)
	if err != nil {
		t.Fatal(err)
	}
	a.cross, err = newCrossSubscriptionManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	a.subTemplates, err = newSubTemplateManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if os.Getenv("UNIFIEDPROXY_DESIGN_ROLE") == "control" {
		a.role = "control"
		m.remote = true
		err = a.initializeFleet(ctx, dir)
	} else {
		a.role = "node" // Production Run defaults to node, even before enrollment.
		err = a.initializeStandalone(ctx, dir)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err = a.peers.configure("Design fixture", "http://"+address); err != nil {
		t.Fatal(err)
	}
	t.Log("READY", address)
	if err = http.ListenAndServe(address, a.routes()); err != nil {
		t.Fatal(err)
	}
}
