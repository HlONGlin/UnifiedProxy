package app

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestCoreControlManualStopSurvivesPolicyAndServiceRestart(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "core")
	if out, err := exec.Command("go", "build", "-o", bin, "./internal/app/testdata/fleetcore").CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v %s", err, out)
	}
	dir := t.TempDir()
	m := &CoreManager{dataDir: dir}
	if err := m.load(); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	m.state.Settings.CorePath, m.state.Settings.APIAddress = bin, address
	n := &nodeRuntime{manager: m, sidecars: &agent.Sidecars{Dir: dir}}
	defer m.stopCore()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	policy := func(revision uint64) fleet.DesiredState {
		state := defaultState()
		state.Settings.LogLevel = "info"
		if revision > 1 {
			state.Settings.LogLevel = "warning"
		}
		model, _ := json.Marshal(state)
		yaml, _ := renderStateConfigAt(state, 0)
		s := fleet.DesiredState{NodeID: "core-control-test", Revision: revision, Model: model, MihomoYAML: string(yaml), Metering: "agent"}
		s.Digest, _ = s.ContentDigest()
		return s
	}
	if err := n.Apply(ctx, policy(1)); err != nil {
		t.Fatal("first policy must still start a new node", err)
	}
	control := func(action string) {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{"action": action})
		if _, err := n.Execute(ctx, fleet.Task{Kind: "core-control", Payload: payload}); err != nil {
			t.Fatal(action, err)
		}
	}
	assertStopped := func() {
		t.Helper()
		if n.Snapshot().Running {
			t.Fatal("manually stopped core is running")
		}
		if conn, err := net.DialTimeout("tcp", address, time.Second); err == nil {
			conn.Close()
			t.Fatal("core listener remained open after stop completed")
		}
	}
	control("stop")
	assertStopped()
	if err := n.Apply(ctx, policy(2)); err != nil {
		t.Fatal("stopped nodes must still accept configuration", err)
	}
	assertStopped()
	if n.applied.Revision != 2 {
		t.Fatal("configuration was not applied while stopped")
	}
	if err := m.restartCore(); err != nil {
		t.Fatal(err)
	}
	assertStopped()
	// The service's new process loads the same local intent before auto-start.
	restarted := &CoreManager{dataDir: dir}
	if err := restarted.load(); err != nil {
		t.Fatal(err)
	}
	defer restarted.stopCore()
	if err := restarted.startCore(); err != nil {
		t.Fatal(err)
	}
	restarted.mu.Lock()
	running := restarted.runningLocked()
	restarted.mu.Unlock()
	if running {
		t.Fatal("service auto-start ignored persistent manual stop")
	}

	health := func(want int) {
		t.Helper()
		a := &App{manager: m, nodeRuntime: n}
		r := httptest.NewRequest("GET", "/fleet/v1/health", nil)
		r.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		a.withFleetTransport(http.NotFoundHandler()).ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("health = %d, want %d: %s", w.Code, want, w.Body.String())
		}
	}
	health(http.StatusOK)
	// A failed explicit start must preserve the old stopped intent.
	m.mu.Lock()
	m.state.Settings.CorePath = filepath.Join(dir, "missing-core")
	m.mu.Unlock()
	if err := m.controlCore("start"); err == nil {
		t.Fatal("missing core was reported as started")
	}
	if stopped, err := m.coreStopped(); err != nil || !stopped {
		t.Fatal("failed start lost manual stop", stopped, err)
	}
	m.mu.Lock()
	m.state.Settings.CorePath = bin
	m.mu.Unlock()
	for i := 0; i < 3; i++ {
		control("start")
		if !n.Snapshot().Running {
			t.Fatal("explicit start did not resume the core")
		}
		control("restart")
		control("stop")
		assertStopped()
	}
	// Unplanned failure still fails upgrade health checks.
	control("start")
	if err := m.stopCore(); err != nil {
		t.Fatal(err)
	}
	health(http.StatusServiceUnavailable)
	if _, err := os.Stat(filepath.Join(dir, "core-stopped")); !os.IsNotExist(err) {
		t.Fatal("internal shutdown changed user intent", err)
	}
}
