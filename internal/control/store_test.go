package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
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
