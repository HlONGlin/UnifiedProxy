package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestFleetCorruptStoreAndBackupFailClosed(t *testing.T) {
	for _, kind := range []string{"null-node", "null-task", "empty-rollout", "bad-desired"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			s, err := Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			n := enrollTest(t, s)
			d := clone(s.disk)
			d.Rollouts = map[string]*Rollout{}
			switch kind {
			case "null-node":
				d.Nodes["bad"] = nil
			case "null-task":
				d.Tasks["bad"] = nil
			case "empty-rollout":
				d.Rollouts["bad"] = &Rollout{ID: "bad", Status: "canary", Nodes: []string{n.ID}}
			case "bad-desired":
				d.Nodes[n.ID].Desired = &fleet.DesiredState{NodeID: n.ID, Revision: 1}
			}
			data, _ := json.Marshal(struct {
				Format   int            `json:"format"`
				Identity fleet.Identity `json:"identity"`
				State    diskState      `json:"state"`
			}{1, s.Identity, d})
			before, _ := s.Backup()
			if err := s.Restore(data); err == nil {
				t.Fatal("corrupt backup accepted")
			}
			after, _ := s.Backup()
			if string(before) != string(after) {
				t.Fatal("rejected backup mutated state")
			}
			if err := fleet.WriteJSON(filepath.Join(dir, "fleet.json"), d); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(dir); err == nil {
				t.Fatal("corrupt store opened")
			}
			if _, err := os.Stat(filepath.Join(dir, "fleet.json")); err != nil {
				t.Fatal(err)
			}
		})
	}
}
