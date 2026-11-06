package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/control"
)

type wipeRuntime struct {
	testRuntime
	wipes atomic.Int32
}

func (r *wipeRuntime) WipeProcesses(context.Context) error {
	r.wipes.Add(1)
	return nil
}

func TestSignedRevokeStopsProcessesAndDeletesNodeData(t *testing.T) {
	store, err := control.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(&control.Server{Store: store})
	defer server.Close()
	token, err := store.IssueToken("node", "")
	if err != nil {
		t.Fatal(err)
	}
	installDir := t.TempDir()
	dir := filepath.Join(installDir, "data")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	cfg, err := Enroll(context.Background(), dir, server.URL, store.Identity.Public(), token, "node")
	if err != nil {
		t.Fatal(err)
	}
	corePath := filepath.Join(installDir, "mihomo")
	outsideCore := filepath.Join(t.TempDir(), "mihomo")
	// corePath points outside the install directory and must be kept. The
	// installer-default binary next to the data directory is still removed.
	state := []byte(`{"settings":{"corePath":"` + filepath.ToSlash(outsideCore) + `"}}`)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), state, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config.yaml", "applied-state.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("secret"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"caddy-naive", "shadow-tls", "ssserver", "caddy-naive.source.json"} {
		if err := os.WriteFile(filepath.Join(dir, "bin", name), []byte("sidecar"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, "vpngate"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"vpnclient", "vpncmd", "hamcore.se2"} {
		if err := os.WriteFile(filepath.Join(dir, "vpngate", name), []byte("vpngate"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(corePath, []byte("mihomo"), 0700); err != nil {
		t.Fatal(err)
	}
	nodeBinary := filepath.Join(installDir, "node")
	if err := os.WriteFile(nodeBinary, []byte("node-program"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsideCore, []byte("system-mihomo"), 0700); err != nil {
		t.Fatal(err)
	}
	keeper := filepath.Join(installDir, "keep-outside-node-data")
	if err := os.WriteFile(keeper, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(keeper)
		_ = os.Remove(corePath)
		_ = os.Remove(nodeBinary)
	})
	if err := store.Revoke(cfg.NodeID); err != nil {
		t.Fatal(err)
	}
	runtime := &wipeRuntime{}
	a, err := New(dir, runtime)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.run(context.Background(), 20*time.Millisecond, func(context.Context) string { return "" })
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("agent kept running after revocation")
	}
	if !a.RevokedLocally() || runtime.wipes.Load() != 1 {
		t.Fatal("revoke did not stop local processes", a.RevokedLocally(), runtime.wipes.Load())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatal("node data remains", entries)
	}
	if data, err := os.ReadFile(keeper); err != nil || string(data) != "keep" {
		t.Fatal("wipe escaped the data directory", err)
	}
	if _, err := os.Stat(corePath); !os.IsNotExist(err) {
		t.Fatal("downloaded Mihomo was kept", err)
	}
	if data, err := os.ReadFile(nodeBinary); err != nil || string(data) != "node-program" {
		t.Fatal("node program was removed", err)
	}
	if data, err := os.ReadFile(outsideCore); err != nil || string(data) != "system-mihomo" {
		t.Fatal("Mihomo outside the install directory was removed", err)
	}
}

func TestUnsignedRejectionDoesNotDeleteNodeData(t *testing.T) {
	store, err := control.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var block atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if block.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"access-revoked","wipe":true}`))
			return
		}
		(&control.Server{Store: store}).ServeHTTP(w, r)
	}))
	defer server.Close()
	token, err := store.IssueToken("node", "")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err = Enroll(context.Background(), dir, server.URL, store.Identity.Public(), token, "node"); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(secret, []byte("keep-me"), 0600); err != nil {
		t.Fatal(err)
	}
	block.Store(true)
	runtime := &wipeRuntime{}
	a, err := New(dir, runtime)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.run(ctx, 20*time.Millisecond, func(context.Context) string { return "" })
	}()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("agent did not stop")
	}
	if a.RevokedLocally() || runtime.wipes.Load() != 0 {
		t.Fatal("unsigned 401 wiped the node")
	}
	data, err := os.ReadFile(secret)
	if err != nil || string(data) != "keep-me" {
		t.Fatal("node data deleted without a signed revocation", err)
	}
}
