package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func enrollTest(t *testing.T, s *Store) *Node {
	t.Helper()
	token, err := s.IssueToken("test", "")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := fleet.NewIdentity()
	n, err := s.Enroll(fleet.Enrollment{Token: token, PublicKey: id.Public()})
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func TestFleetTokenSingleUseExpiryAndHashOnly(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.IssueToken("test", "")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "fleet.json"))
	if strings.Contains(string(b), token) {
		t.Fatal("plaintext token persisted")
	}
	id, _ := fleet.NewIdentity()
	if _, err = s.Enroll(fleet.Enrollment{Token: token, PublicKey: id.Public()}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Enroll(fleet.Enrollment{Token: token, PublicKey: id.Public()}); err == nil {
		t.Fatal("consumed token accepted")
	}
	token, err = s.IssueToken("expired", "")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	v := s.disk.Tokens[tokenHash(token)]
	v.ExpiresAt = time.Now().Add(-time.Minute)
	s.disk.Tokens[tokenHash(token)] = v
	s.mu.Unlock()
	if _, err = s.Enroll(fleet.Enrollment{Token: token, PublicKey: id.Public()}); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestRevokedNodeRequiresExplicitRestoreBeforeTokenIssue(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	n := enrollTest(t, s)
	if err := s.Revoke(n.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssueToken("blocked", n.ID); err == nil {
		t.Fatal("revoked node received an enrollment token")
	}
	if err := s.RestoreNode(n.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssueToken("restored", n.ID); err != nil {
		t.Fatalf("restored node could not receive a token: %v", err)
	}
}

func TestDeleteRevokedNodesIsAtomicAndRemovesNodeScopedData(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	active, err := s.AddNode("active", "active.example")
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := s.AddNode("revoked", "revoked.example")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.IssueToken("pending", revoked.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Enqueue(revoked.ID, "collect-status", "delete-node", nil, time.Hour); err != nil {
		t.Fatal(err)
	}
	for _, node := range []*Node{active, revoked} {
		if err = s.UpdateStatus(node.ID, fleet.Status{Capabilities: []string{"collect-status"}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.Rollout([]string{active.ID, revoked.ID}, "collect-status", nil); err != nil {
		t.Fatal(err)
	}
	if err = s.Revoke(revoked.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteRevokedNodes([]string{active.ID, revoked.ID}); err == nil {
		t.Fatal("mixed active and revoked deletion was accepted")
	}
	if s.Node(active.ID) == nil || s.Node(revoked.ID) == nil {
		t.Fatal("failed batch deletion changed the store")
	}
	if err = s.DeleteRevokedNodes([]string{revoked.ID}); err != nil {
		t.Fatal(err)
	}
	if s.Node(revoked.ID) != nil || s.Node(active.ID) == nil {
		t.Fatal("node deletion affected the wrong records")
	}
	rollouts := s.Rollouts()
	if len(rollouts) != 1 || rollouts[0].Status != "failed" || len(rollouts[0].Nodes) != 1 || rollouts[0].Nodes[0] != active.ID {
		t.Fatal("rollout history was not kept valid after node deletion", rollouts)
	}
	s.mu.Lock()
	for _, task := range s.disk.Tasks {
		if task.NodeID == revoked.ID {
			s.mu.Unlock()
			t.Fatal("deleted node task was retained")
		}
	}
	for _, token := range s.disk.Tokens {
		if token.NodeID == revoked.ID {
			s.mu.Unlock()
			t.Fatal("deleted node token was retained")
		}
	}
	s.mu.Unlock()
	if _, err = Open(dir); err != nil {
		t.Fatalf("store became invalid after deletion: %v", err)
	}
}

func TestFleetTaskLeaseIdempotencyAndRestart(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := enrollTest(t, s)
	if err = s.UpdateStatus(n.ID, fleet.Status{Capabilities: []string{"collect-status"}}); err != nil {
		t.Fatal(err)
	}
	task, err := s.Enqueue(n.ID, "collect-status", "one", nil, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	same, err := s.Enqueue(n.ID, "collect-status", "one", nil, time.Hour)
	if err != nil || same.ID != task.ID {
		t.Fatal("not idempotent", err)
	}
	if _, err = s.Enqueue(n.ID, "collect-status", "one", json.RawMessage(`{}`), time.Hour); err == nil {
		t.Fatal("conflicting key accepted")
	}
	now := time.Now()
	claimed, err := s.Claim(n.ID, now)
	if err != nil || len(claimed) != 1 {
		t.Fatal(claimed, err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err = s.Claim(n.ID, now.Add(time.Minute))
	if err != nil || len(claimed) != 0 {
		t.Fatal("lease lost after restart")
	}
	for i := 1; i <= 2; i++ {
		claimed, err = s.Claim(n.ID, now.Add(time.Duration(i)*3*time.Minute))
		if err != nil || len(claimed) != 1 {
			t.Fatal("retry missing", err)
		}
	}
	claimed, err = s.Claim(n.ID, now.Add(10*time.Minute))
	if err != nil || len(claimed) != 0 {
		t.Fatal("more than 3 deliveries")
	}
	if s.Tasks()[0].Status != "failed" {
		t.Fatal("receipt timeout not recorded")
	}
}
func TestFleetExpiredTasksAndCapabilityNegotiation(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	n := enrollTest(t, s)
	if _, err = s.Enqueue(n.ID, "geo-update", "one", nil, time.Second); err != nil {
		t.Fatal(err)
	}
	claimed, err := s.Claim(n.ID, time.Now())
	if err != nil || len(claimed) != 0 {
		t.Fatal("unsupported task delivered")
	}
	if !s.Node(n.ID).NeedsUpgrade {
		t.Fatal("upgrade badge missing")
	}
	if err = s.UpdateStatus(n.ID, fleet.Status{Capabilities: []string{"geo-update"}}); err != nil {
		t.Fatal(err)
	}
	claimed, err = s.Claim(n.ID, time.Now().Add(2*time.Second))
	if err != nil || len(claimed) != 0 {
		t.Fatal("expired task delivered")
	}
}
func TestFleetDesiredRevisionAndCanaryGate(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, b := enrollTest(t, s), enrollTest(t, s)
	state := fleet.DesiredState{MihomoYAML: "mode: rule", Model: json.RawMessage(`{}`)}
	first, err := s.SetDesired(a.ID, state)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.SetDesired(a.ID, state)
	if err != nil || second.Revision != first.Revision {
		t.Fatal("unchanged state incremented revision")
	}
	state.MihomoYAML = "mode: direct"
	next, err := s.SetDesired(a.ID, state)
	if err != nil || next.Revision != 2 {
		t.Fatal("revision is not monotonic")
	}
	for _, n := range []*Node{a, b} {
		if err = s.UpdateStatus(n.ID, fleet.Status{Capabilities: []string{"geo-update"}}); err != nil {
			t.Fatal(err)
		}
	}
	roll, err := s.Rollout([]string{a.ID, b.ID}, "geo-update", nil)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.Claim(b.ID, time.Now())
	if err != nil || len(claimed) != 0 {
		t.Fatal("second node ran before canary")
	}
	claimed, err = s.Claim(a.ID, time.Now())
	if err != nil || len(claimed) != 1 {
		t.Fatal(err)
	}
	if err = s.Result(a.ID, fleet.Result{TaskID: claimed[0].ID, Status: "failed"}); err != nil {
		t.Fatal(err)
	}
	if s.Rollouts()[0].ID != roll.ID || s.Rollouts()[0].Status != "failed" {
		t.Fatal("failed canary did not stop rollout")
	}
	claimed, err = s.Claim(b.ID, time.Now())
	if err != nil || len(claimed) != 0 {
		t.Fatal("failed canary released next node")
	}
}
