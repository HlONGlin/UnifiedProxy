package agent

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/control"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type testRuntime struct {
	mu       sync.Mutex
	calls    int
	revision uint64
}

func (r *testRuntime) Snapshot() fleet.Status {
	return fleet.Status{Version: "test", Protocol: 1, Capabilities: []string{"collect-status", "apply-state"}}
}
func (r *testRuntime) Apply(_ context.Context, s fleet.DesiredState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.revision = s.Revision
	return nil
}
func (r *testRuntime) Execute(_ context.Context, _ fleet.Task) (json.RawMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return json.RawMessage(`{"running":true}`), nil
}
func TestFleetThreeNodesEnrollPollExecuteAndRestartReceipt(t *testing.T) {
	store, err := control.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(&control.Server{Store: store})
	defer server.Close()
	for i := 0; i < 3; i++ {
		token, err := store.IssueToken("node", "")
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		cfg, err := Enroll(context.Background(), dir, server.URL, store.Identity.Public(), token, "node")
		if err != nil {
			t.Fatal(err)
		}
		runtime := &testRuntime{}
		a, err := New(dir, runtime)
		if err != nil {
			t.Fatal(err)
		}
		if err = store.UpdateStatus(cfg.NodeID, runtime.Snapshot()); err != nil {
			t.Fatal(err)
		}
		task, err := store.Enqueue(cfg.NodeID, "collect-status", "status", nil, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		var poll fleet.PollResponse
		if err = a.Client.request(context.Background(), "/fleet/v1/poll", cfg.NodeID, fleet.Poll{Status: runtime.Snapshot()}, &poll); err != nil {
			t.Fatal(err)
		}
		if len(poll.Tasks) != 1 {
			t.Fatal("missing task")
		}
		result := a.Process(context.Background(), poll.Tasks[0])
		if result.Status != "success" {
			t.Fatal(result)
		}
		a, err = New(dir, runtime)
		if err != nil {
			t.Fatal(err)
		}
		result = a.Process(context.Background(), *task)
		if result.Status != "success" || runtime.calls != 1 {
			t.Fatal("duplicate execution after restart", result)
		}
		if err = a.acknowledge(context.Background(), result); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.Nodes()) != 3 {
		t.Fatal("expected three independent identities")
	}
}
func TestFleetWrongControlKeyRejected(t *testing.T) {
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
	other, _ := fleet.NewIdentity()
	if _, err = Enroll(context.Background(), t.TempDir(), server.URL, other.Public(), token, "node"); err == nil {
		t.Fatal("wrong controller key accepted")
	}
}

func TestFleetCorruptReceiptJournalRejectedWithoutMutation(t *testing.T) {
	store, err := control.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(&control.Server{Store: store})
	defer server.Close()
	token, err := store.IssueToken("fixture", "")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err = Enroll(context.Background(), dir, server.URL, store.Identity.Public(), token, "fixture"); err != nil {
		t.Fatal(err)
	}
	valid := Receipt{Key: "key", Kind: "collect-status", ExpiresAt: time.Now().Add(time.Minute)}
	for _, kind := range []string{"null-entry", "missing-key", "task-mismatch", "invalid-status"} {
		t.Run(kind, func(t *testing.T) {
			r := valid
			journal := Journal{Receipts: map[string]*Receipt{"task": &r}}
			switch kind {
			case "null-entry":
				journal.Receipts["task"] = nil
			case "missing-key":
				r.Key = ""
			case "task-mismatch":
				r.Result = &fleet.Result{TaskID: "other", Status: "success"}
			case "invalid-status":
				r.Result = &fleet.Result{TaskID: "task", Status: "arbitrary"}
			}
			path := filepath.Join(dir, "agent-receipts.json")
			if err := fleet.WriteJSON(path, journal); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(path)
			if _, err := New(dir, &testRuntime{}); err == nil {
				t.Fatal("corrupt receipt journal accepted")
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("rejected journal was modified")
			}
		})
	}
}
