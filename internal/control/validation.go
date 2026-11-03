package control

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
)

// Validate disk input before publishing it to readers. In particular, null
// map entries and empty active rollouts must never reach the queue walkers.
func validateDiskState(d diskState) error {
	if d.Format != 1 || d.Nodes == nil || d.Tasks == nil || d.Tokens == nil {
		return errors.New("invalid fleet store")
	}
	for id, n := range d.Nodes {
		if n == nil || !fleet.SafeName(id) || n.ID != id {
			return fmt.Errorf("invalid fleet node %q", id)
		}
		if n.PublicKey != "" {
			key, err := base64.StdEncoding.DecodeString(n.PublicKey)
			if err != nil || len(key) != 32 {
				return fmt.Errorf("invalid node identity %q", id)
			}
		}
		if n.Desired != nil {
			if n.Desired.NodeID != id || n.Desired.Validate() != nil {
				return fmt.Errorf("invalid desired state %q", id)
			}
		}
	}
	for id, task := range d.Tasks {
		if task == nil || !fleet.SafeName(id) || task.ID != id || d.Nodes[task.NodeID] == nil || task.IdempotencyKey == "" || task.Attempts < 0 || task.ExpiresAt.Before(task.CreatedAt) {
			return fmt.Errorf("invalid fleet task %q", id)
		}
		if len(task.Payload) > 0 && !json.Valid(task.Payload) {
			return fmt.Errorf("invalid task payload %q", id)
		}
		switch task.Status {
		case "pending", "leased", "success", "failed", "expired", "cancelled", "superseded", "unsupported":
		default:
			return fmt.Errorf("invalid task status %q", id)
		}
		if task.Result != nil && task.Result.TaskID != id {
			return fmt.Errorf("invalid task receipt %q", id)
		}
	}
	for hash, token := range d.Tokens {
		if len(hash) != 64 || token.Hash != hash || (token.NodeID != "" && d.Nodes[token.NodeID] == nil) {
			return errors.New("invalid enrollment token record")
		}
	}
	for id, r := range d.Rollouts {
		if r == nil || !fleet.SafeName(id) || r.ID != id || len(r.Nodes) == 0 || len(r.Tasks) == 0 || len(r.Tasks) > len(r.Nodes) {
			return fmt.Errorf("invalid rollout %q", id)
		}
		switch r.Status {
		case "canary", "running", "success", "failed":
		default:
			return fmt.Errorf("invalid rollout status %q", id)
		}
		seen := map[string]bool{}
		for _, node := range r.Nodes {
			if d.Nodes[node] == nil || seen[node] {
				return fmt.Errorf("invalid rollout node %q", id)
			}
			seen[node] = true
		}
		for i, taskID := range r.Tasks {
			t := d.Tasks[taskID]
			// Completed task history can be pruned after retention expires.
			if t == nil && (r.Status == "success" || r.Status == "failed") {
				continue
			}
			if t == nil || t.NodeID != r.Nodes[i] || t.Kind != r.Kind {
				return fmt.Errorf("invalid rollout task %q", id)
			}
		}
	}
	return nil
}
