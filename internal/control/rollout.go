package control

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
)

type Rollout struct {
	ID        string          `json:"id"`
	Nodes     []string        `json:"nodes"`
	Kind      string          `json:"kind"`
	Payload   json.RawMessage `json:"payload"`
	Tasks     []string        `json:"tasks"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (s *Store) Rollout(nodes []string, kind string, payload json.RawMessage) (*Rollout, error) {
	if len(nodes) == 0 || len(nodes) > 500 {
		return nil, errors.New("rollout requires 1-500 nodes")
	}
	if !fleet.Supports([]string{"kernel-switch", "geo-update", "cert-renew", "agent-upgrade", "collect-status", "test-egress"}, kind) {
		return nil, errors.New("invalid rollout task kind")
	}
	var out *Rollout
	err := s.change(func(d *diskState) error {
		seen := map[string]bool{}
		for _, id := range nodes {
			n := d.Nodes[id]
			if n == nil || n.Revoked || seen[id] {
				return errors.New("unknown, revoked, or duplicate rollout node")
			}
			if !fleet.Supports(n.Status.Capabilities, kind) {
				return errors.New("rollout node does not support task")
			}
			seen[id] = true
		}
		r := &Rollout{ID: fleet.ID(), Nodes: nodes, Kind: kind, Payload: payload, Status: "canary", CreatedAt: time.Now().UTC()}
		task, err := queue(d, nodes[0], kind, r.ID+":"+nodes[0], payload, 30*time.Minute)
		if err != nil {
			return err
		}
		r.Tasks = append(r.Tasks, task.ID)
		if d.Rollouts == nil {
			d.Rollouts = map[string]*Rollout{}
		}
		d.Rollouts[r.ID] = r
		out = clone(r)
		return nil
	})
	if err == nil {
		s.notify()
	}
	return out, err
}

// A failed or expired canary never releases the remaining hosts. Once the
// canary succeeds, later hosts execute serially with a durable receipt gate.
func advanceRollouts(d *diskState) {
	for _, r := range d.Rollouts {
		if r.Status != "canary" && r.Status != "running" {
			continue
		}
		if len(r.Tasks) == 0 || len(r.Nodes) == 0 || len(r.Tasks) > len(r.Nodes) {
			r.Status = "failed"
			continue
		}
		last := d.Tasks[r.Tasks[len(r.Tasks)-1]]
		if last == nil {
			r.Status = "failed"
			continue
		}
		if last.Status == "pending" || last.Status == "leased" {
			continue
		}
		if last.Status != "success" {
			r.Status = "failed"
			continue
		}
		if len(r.Tasks) == len(r.Nodes) {
			r.Status = "success"
			continue
		}
		id := r.Nodes[len(r.Tasks)]
		task, err := queue(d, id, r.Kind, r.ID+":"+id, r.Payload, 30*time.Minute)
		if err != nil {
			r.Status = "failed"
			continue
		}
		r.Tasks = append(r.Tasks, task.ID)
		r.Status = "running"
	}
}
func (s *Store) Rollouts() []*Rollout {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []*Rollout{}
	for _, r := range s.disk.Rollouts {
		out = append(out, clone(r))
	}
	return out
}
