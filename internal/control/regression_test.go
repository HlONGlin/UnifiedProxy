package control

import (
	"encoding/json"
	"github.com/RomanovCaesar/m-ui/internal/fleet"
	"testing"
	"time"
)

func TestFleetDesiredRetryOfflineExpiryAndRebinding(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	n := enrollTest(t, s)
	state, err := s.SetDesired(n.ID, fleet.DesiredState{Model: json.RawMessage("{}"), MihomoYAML: "mode: rule"})
	if err != nil {
		t.Fatal(err)
	}
	status := fleet.Status{Protocol: 1, Capabilities: []string{"apply-state"}}
	if err = s.UpdateStatus(n.ID, status); err != nil {
		t.Fatal(err)
	}
	tasks, err := s.Claim(n.ID, time.Now(), n.PublicKey)
	if err != nil || len(tasks) != 1 {
		t.Fatal(tasks, err)
	}
	first := tasks[0]
	if err = s.Result(n.ID, fleet.Result{TaskID: first.ID, Status: "failed"}, n.PublicKey); err != nil {
		t.Fatal(err)
	}
	retry, err := s.SetDesired(n.ID, *state)
	if err != nil || retry.Revision != state.Revision {
		t.Fatal("retry changed policy revision", err)
	}
	tasks, err = s.Claim(n.ID, time.Now(), n.PublicKey)
	if err != nil || len(tasks) != 1 || tasks[0].ID == first.ID {
		t.Fatal("failed task was resurrected instead of creating a new delivery", err)
	}
	expired := tasks[0].ID
	s.mu.Lock()
	s.disk.Tasks[expired].ExpiresAt = time.Now().Add(-time.Minute)
	s.mu.Unlock()
	if err = s.UpdateStatus(n.ID, status); err != nil {
		t.Fatal(err)
	}
	tasks, err = s.Claim(n.ID, time.Now(), n.PublicKey)
	if err != nil || len(tasks) != 1 || tasks[0].ID == expired {
		t.Fatal("offline expiry prevented reconciliation", err)
	}
	key, _ := fleet.NewIdentity()
	token, err := s.IssueToken("rebind", n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Enroll(fleet.Enrollment{Token: token, PublicKey: key.Public()}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Claim(n.ID, time.Now(), n.PublicKey); err == nil {
		t.Fatal("old identity claimed tasks after rebind")
	}
	if err = s.UpdateStatus(n.ID, status, n.PublicKey); err == nil {
		t.Fatal("old identity reported after rebind")
	}
	if err = s.Result(n.ID, fleet.Result{TaskID: tasks[0].ID, Status: "success"}, n.PublicKey); err == nil {
		t.Fatal("old identity acknowledged after rebind")
	}
	if err = s.UpdateStatus(n.ID, status, key.Public()); err != nil {
		t.Fatal(err)
	}
	tasks, err = s.Claim(n.ID, time.Now(), key.Public())
	if err != nil || len(tasks) != 1 {
		t.Fatal("new identity did not receive desired state", err)
	}
}
