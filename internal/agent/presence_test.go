package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/control"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type busyRuntime struct {
	mu      sync.Mutex
	started chan struct{}
	calls   atomic.Int32
}

func (r *busyRuntime) Snapshot() fleet.Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fleet.Status{Version: "test", Protocol: 1, Revision: 7, Capabilities: []string{"collect-status"}}
}
func (r *busyRuntime) Apply(context.Context, fleet.DesiredState) error { return nil }
func (r *busyRuntime) Execute(ctx context.Context, _ fleet.Task) (json.RawMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls.Add(1)
	close(r.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestHeartbeatSurvivesBusyRuntimeAndStopsOnCancel(t *testing.T) {
	s, err := control.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(&control.Server{Store: s})
	defer server.Close()
	token, _ := s.IssueToken("node", "")
	dir := t.TempDir()
	cfg, err := Enroll(context.Background(), dir, server.URL, s.Identity.Public(), token, "node")
	if err != nil {
		t.Fatal(err)
	}
	runtime := &busyRuntime{started: make(chan struct{})}
	a, err := New(dir, runtime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Enqueue(cfg.NodeID, "collect-status", "busy", nil, time.Minute); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.run(ctx, 20*time.Millisecond, func(context.Context) string { return "1.1.1.1" })
	}()
	select {
	case <-runtime.started:
	case <-time.After(3 * time.Second):
		t.Fatal("task not started")
	}
	before := s.Node(cfg.NodeID).LastSeen
	deadline := time.Now().Add(3 * time.Second)
	for {
		n := s.Node(cfg.NodeID)
		if n.LastSeen.After(before) && n.SeenIP == "1.1.1.1" {
			if n.Status.Revision != 7 {
				t.Fatal("heartbeat overwrote snapshot")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("busy task blocked heartbeat")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("agent did not stop")
	}
	last := s.Node(cfg.NodeID).LastSeen
	time.Sleep(60 * time.Millisecond)
	if !s.Node(cfg.NodeID).LastSeen.Equal(last) || runtime.calls.Load() != 1 {
		t.Fatal("heartbeat leaked or task ran twice")
	}
}

func TestHeartbeatStopsAfterDetach(t *testing.T) {
	for _, mode := range []string{"leave", "break-glass"} {
		t.Run(mode, func(t *testing.T) {
			s, err := control.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(&control.Server{Store: s})
			defer server.Close()
			token, _ := s.IssueToken("node", "")
			dir := t.TempDir()
			cfg, err := Enroll(context.Background(), dir, server.URL, s.Identity.Public(), token, "node")
			if err != nil {
				t.Fatal(err)
			}
			a, err := New(dir, &testRuntime{})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan struct{})
			go func() { defer close(done); a.heartbeatLoop(ctx, 20*time.Millisecond) }()
			deadline := time.Now().Add(3 * time.Second)
			for s.Node(cfg.NodeID).LastSeen.IsZero() {
				if time.Now().After(deadline) {
					t.Fatal("missing heartbeat")
				}
				time.Sleep(10 * time.Millisecond)
			}
			if mode == "leave" {
				err = os.Remove(filepath.Join(dir, ConfigFile))
			} else {
				err = os.WriteFile(filepath.Join(dir, "break-glass"), nil, 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("detached heartbeat continued")
			}
		})
	}
}

func TestPublicIPProbeFallbackValidationAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/html":
			w.Write([]byte("<html>1.1.1.1</html>"))
		case "/private":
			w.Write([]byte("192.168.0.1"))
		case "/oversize":
			w.Write([]byte(strings.Repeat(" ", 129) + "1.1.1.1"))
		case "/error":
			w.WriteHeader(500)
			w.Write([]byte("1.1.1.1"))
		case "/slow":
			<-r.Context().Done()
		default:
			w.Write([]byte("\n2606:4700:4700::1111\n"))
		}
	}))
	defer server.Close()
	client := &http.Client{Timeout: 40 * time.Millisecond}
	providers := []string{}
	for _, path := range []string{"/html", "/private", "/oversize", "/error", "/slow", "/good"} {
		providers = append(providers, server.URL+path)
	}
	if got := probePublicIP(context.Background(), client, providers); got != "2606:4700:4700::1111" {
		t.Fatal("fallback failed", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := probePublicIP(ctx, client, providers); got != "" {
		t.Fatal("cancel ignored", got)
	}
}
