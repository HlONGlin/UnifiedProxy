// Package control implements the fleet registry and durable task queue.
package control

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type Node struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Address      string              `json:"address"`
	PublicKey    string              `json:"publicKey,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
	LastSeen     time.Time           `json:"lastSeen"`
	Revoked      bool                `json:"revoked"`
	NeedsUpgrade bool                `json:"needsUpgrade"`
	Status       fleet.Status        `json:"status"`
	Desired      *fleet.DesiredState `json:"desired,omitempty"`
}
type Token struct {
	Hash      string    `json:"hash"`
	NodeID    string    `json:"nodeId,omitempty"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type Audit struct {
	At     time.Time `json:"at"`
	Action string    `json:"action"`
	Target string    `json:"target"`
}
type diskState struct {
	Rollouts map[string]*Rollout    `json:"rollouts,omitempty"`
	Format   int                    `json:"format"`
	Nodes    map[string]*Node       `json:"nodes"`
	Tasks    map[string]*fleet.Task `json:"tasks"`
	Tokens   map[string]Token       `json:"tokens"`
	Audit    []Audit                `json:"audit"`
}
type Store struct {
	mu       sync.Mutex
	path     string
	disk     diskState
	wake     chan struct{}
	Identity fleet.Identity
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	id, err := fleet.LoadIdentityWithLegacy(filepath.Join(dir, "control-identity.json"), filepath.Join(dir, "multi-control.json"))
	if err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "fleet.json"), Identity: id, wake: make(chan struct{})}
	err = fleet.ReadJSON(s.path, &s.disk)
	if errors.Is(err, os.ErrNotExist) {
		s.disk = diskState{Format: 1, Nodes: map[string]*Node{}, Tasks: map[string]*fleet.Task{}, Tokens: map[string]Token{}}
		err = fleet.WriteJSON(s.path, s.disk)
	}
	if err != nil {
		return nil, err
	}
	if err := validateDiskState(s.disk); err != nil {
		return nil, err
	}
	if s.disk.Rollouts == nil {
		s.disk.Rollouts = map[string]*Rollout{}
	}
	return s, nil
}

func clone[T any](v T) T {
	var out T
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}
func (s *Store) change(fn func(*diskState) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.disk)
	if err := fn(&next); err != nil {
		return err
	}
	serialized, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if len(serialized) > 64<<20 {
		return errors.New("fleet state exceeds 64 MiB limit; export and prune completed tasks")
	}
	if err := fleet.WriteFile(s.path, serialized, 0600); err != nil {
		return err
	}
	s.disk = next
	return nil
}
func (s *Store) notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.wake)
	s.wake = make(chan struct{})
}
func audit(d *diskState, action, target string) {
	d.Audit = append(d.Audit, Audit{time.Now().UTC(), action, target})
	if len(d.Audit) > 1000 {
		d.Audit = d.Audit[len(d.Audit)-1000:]
	}
}
func tokenHash(t string) string { h := sha256.Sum256([]byte(t)); return hex.EncodeToString(h[:]) }

func (s *Store) IssueToken(name, nodeID string) (string, error) {
	token := fleet.ID() + fleet.ID()
	err := s.change(func(d *diskState) error {
		if len(name) > 200 {
			return errors.New("node name too long")
		}
		if nodeID != "" && d.Nodes[nodeID] == nil {
			return errors.New("unknown target node")
		}
		for k, v := range d.Tokens {
			if time.Now().After(v.ExpiresAt) {
				delete(d.Tokens, k)
			}
		}
		if len(d.Tokens) >= 1000 {
			return errors.New("too many enrollment tokens")
		}
		h := tokenHash(token)
		d.Tokens[h] = Token{Hash: h, Name: name, NodeID: nodeID, ExpiresAt: time.Now().Add(15 * time.Minute)}
		audit(d, "token-issued", nodeID)
		return nil
	})
	return token, err
}
func (s *Store) RevokeToken(token string) error {
	return s.change(func(d *diskState) error {
		delete(d.Tokens, tokenHash(token))
		audit(d, "token-revoked", "")
		return nil
	})
}

func (s *Store) Enroll(e fleet.Enrollment) (*Node, error) {
	pub, err := base64.StdEncoding.DecodeString(e.PublicKey)
	if err != nil || len(pub) != 32 {
		return nil, errors.New("invalid node public key")
	}
	var out *Node
	err = s.change(func(d *diskState) error {
		h := tokenHash(e.Token)
		t, ok := d.Tokens[h]
		if !ok || time.Now().After(t.ExpiresAt) {
			return errors.New("expired or consumed enrollment token")
		}
		for _, n := range d.Nodes {
			if n.PublicKey == e.PublicKey && n.ID != t.NodeID {
				return errors.New("identity already enrolled; issue a rebind token for the existing node")
			}
		}
		id := t.NodeID
		if id == "" {
			id = fleet.ID()
			d.Nodes[id] = &Node{ID: id, Name: t.Name, CreatedAt: time.Now().UTC()}
		}
		n := d.Nodes[id]
		if n == nil {
			return errors.New("target node no longer exists")
		}
		if n.PublicKey != e.PublicKey {
			n.Status = fleet.Status{}
			for _, task := range d.Tasks {
				if task.NodeID == id && (task.Status == "pending" || task.Status == "leased") {
					task.Status = "cancelled"
				}
			}
		}
		n.PublicKey = e.PublicKey
		n.Revoked = false
		n.LastSeen = time.Now().UTC()
		if n.Name == "" {
			n.Name = e.Name
		}
		if n.Name == "" {
			n.Name = id[:12]
		}
		delete(d.Tokens, h)
		audit(d, "node-enrolled", id)
		out = clone(n)
		return nil
	})
	if err == nil {
		s.notify()
	}
	return out, err
}
func (s *Store) AddNode(name, address string) (*Node, error) {
	if len(name) > 200 || len(address) > 500 {
		return nil, errors.New("node metadata too long")
	}
	n := &Node{ID: fleet.ID(), Name: name, Address: address, CreatedAt: time.Now().UTC()}
	err := s.change(func(d *diskState) error { d.Nodes[n.ID] = clone(n); audit(d, "node-created", n.ID); return nil })
	return n, err
}
func (s *Store) UpdateNode(id, name, address string) error {
	return s.change(func(d *diskState) error {
		n := d.Nodes[id]
		if n == nil {
			return errors.New("unknown node")
		}
		if len(name) > 200 || len(address) > 500 {
			return errors.New("node metadata too long")
		}
		n.Name = name
		n.Address = address
		audit(d, "node-updated", id)
		return nil
	})
}
func (s *Store) Revoke(id string) error {
	err := s.change(func(d *diskState) error {
		n := d.Nodes[id]
		if n == nil {
			return errors.New("unknown node")
		}
		n.Revoked = true
		for _, t := range d.Tasks {
			if t.NodeID == id && (t.Status == "pending" || t.Status == "leased") {
				t.Status = "cancelled"
			}
		}
		audit(d, "node-revoked", id)
		return nil
	})
	if err == nil {
		s.notify()
	}
	return err
}
func (s *Store) Node(id string) *Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(s.disk.Nodes[id])
}
func (s *Store) Nodes() []*Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Node, 0, len(s.disk.Nodes))
	for _, n := range s.disk.Nodes {
		out = append(out, clone(n))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (s *Store) Tasks() []*fleet.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*fleet.Task, 0, len(s.disk.Tasks))
	for _, t := range s.disk.Tasks {
		out = append(out, clone(t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}
func (s *Store) Audits() []Audit { s.mu.Lock(); defer s.mu.Unlock(); return clone(s.disk.Audit) }

func queue(d *diskState, id, kind, key string, payload json.RawMessage, ttl time.Duration) (*fleet.Task, error) {
	n := d.Nodes[id]
	if n == nil || n.Revoked {
		return nil, errors.New("unknown or revoked node")
	}
	if key == "" || len(key) > 200 {
		return nil, errors.New("idempotency key required (max 200 bytes)")
	}
	for _, t := range d.Tasks {
		if t.NodeID == id && t.IdempotencyKey == key {
			if t.Kind != kind || string(t.Payload) != string(payload) {
				return nil, errors.New("idempotency key conflicts with existing task")
			}
			return t, nil
		}
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		return nil, errors.New("task lifetime must be between zero and 24 hours")
	}
	if len(d.Tasks) >= 10000 {
		retained := map[string]bool{}
		for _, rollout := range d.Rollouts {
			if rollout.Status == "canary" || rollout.Status == "running" {
				for _, task := range rollout.Tasks {
					retained[task] = true
				}
			}
		}
		for k, t := range d.Tasks {
			if !retained[k] && t.Status != "pending" && t.Status != "leased" && time.Since(t.ExpiresAt) > 24*time.Hour {
				delete(d.Tasks, k)
			}
		}
		if len(d.Tasks) >= 10000 {
			return nil, errors.New("task queue full")
		}
	}
	t := &fleet.Task{ID: fleet.ID(), NodeID: id, Kind: kind, IdempotencyKey: key, Payload: payload, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(ttl), Status: "pending"}
	d.Tasks[t.ID] = t
	audit(d, "task-queued:"+kind, id)
	return t, nil
}

func (s *Store) Enqueue(id, kind, key string, payload json.RawMessage, ttl time.Duration) (*fleet.Task, error) {
	if !fleet.Supports(fleet.TaskKinds, kind) {
		return nil, errors.New("unknown task kind")
	}
	var out *fleet.Task
	err := s.change(func(d *diskState) error { t, err := queue(d, id, kind, key, payload, ttl); out = clone(t); return err })
	if err == nil {
		s.notify()
	}
	return out, err
}
func (s *Store) SetDesired(id string, state fleet.DesiredState) (*fleet.DesiredState, error) {
	var out *fleet.DesiredState
	err := s.change(func(d *diskState) error {
		n := d.Nodes[id]
		if n == nil || n.Revoked {
			return errors.New("unknown or revoked node")
		}
		state.NodeID = id
		state.Metering = "agent"
		state.Revision = 1
		var err error
		state.Digest, err = state.ContentDigest()
		if err != nil {
			return err
		}
		if n.Desired != nil {
			if n.Desired.Digest == state.Digest {
				out = clone(n.Desired)
				return reconcileDesired(d, id, true)
			}
			state.Revision = n.Desired.Revision + 1
		}
		if err = state.Validate(); err != nil {
			return err
		}
		for _, t := range d.Tasks {
			if t.NodeID == id && t.Kind == "apply-state" && (t.Status == "pending" || t.Status == "leased") {
				t.Status = "superseded"
			}
		}
		payload, _ := json.Marshal(state)
		if _, err = queue(d, id, "apply-state", fmt.Sprintf("state:%d:%s", state.Revision, state.Digest), payload, 24*time.Hour); err != nil {
			return err
		}
		n.Desired = clone(&state)
		out = clone(&state)
		audit(d, "desired-state", id)
		return nil
	})
	if err == nil {
		s.notify()
	}
	return out, err
}

// reconcileDesired creates a fresh task after expiry or explicit retry. A task
// ID is never resurrected: nodes retain receipts for their original lifetime.
func reconcileDesired(d *diskState, id string, retry bool) error {
	n := d.Nodes[id]
	if n.Desired == nil || (n.Status.Revision == n.Desired.Revision && n.Status.Digest == n.Desired.Digest) {
		return nil
	}
	var latest *fleet.Task
	for _, task := range d.Tasks {
		if task.NodeID != id || task.Kind != "apply-state" {
			continue
		}
		var desired fleet.DesiredState
		if json.Unmarshal(task.Payload, &desired) != nil || desired.Digest != n.Desired.Digest || desired.Revision != n.Desired.Revision {
			continue
		}
		if (task.Status == "pending" || task.Status == "leased") && time.Now().Before(task.ExpiresAt) {
			return nil
		}
		if latest == nil || task.CreatedAt.After(latest.CreatedAt) {
			latest = task
		}
	}
	if latest != nil && !retry && (latest.Status == "failed" || latest.Status == "success" || latest.Status == "unsupported") {
		return nil
	}
	payload, err := json.Marshal(n.Desired)
	if err != nil {
		return err
	}
	_, err = queue(d, id, "apply-state", fmt.Sprintf("state:%d:%s:%s", n.Desired.Revision, n.Desired.Digest, fleet.ID()), payload, 24*time.Hour)
	return err
}
func checkIdentity(n *Node, key []string) error {
	if n == nil || n.Revoked || (len(key) > 0 && n.PublicKey != key[0]) {
		return errors.New("unknown, revoked, or rebound node")
	}
	return nil
}
func (s *Store) UpdateStatus(id string, status fleet.Status, key ...string) error {
	return s.change(func(d *diskState) error {
		n := d.Nodes[id]
		if err := checkIdentity(n, key); err != nil {
			return err
		}
		if len(status.Capabilities) > 50 || len(status.Traffic) > 50000 {
			return errors.New("oversized status")
		}
		n.Status = status
		n.NeedsUpgrade = false
		n.LastSeen = time.Now().UTC()
		return reconcileDesired(d, id, false)
	})
}

// Claim persists a lease before handing out a task, including after restarts.
func (s *Store) Claim(id string, now time.Time, key ...string) ([]fleet.Task, error) {
	out := []fleet.Task{}
	err := s.change(func(d *diskState) error {
		n := d.Nodes[id]
		if err := checkIdentity(n, key); err != nil {
			return err
		}
		ordered := make([]*fleet.Task, 0)
		for _, t := range d.Tasks {
			if t.NodeID == id {
				ordered = append(ordered, t)
			}
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })
		for _, t := range ordered {
			if t.Status != "pending" && t.Status != "leased" {
				continue
			}
			if !now.Before(t.ExpiresAt) {
				t.Status = "expired"
				continue
			}
			if t.Status == "leased" && now.Before(t.LeaseUntil) {
				continue
			}
			if t.Attempts >= 3 {
				t.Status = "failed"
				t.Result = &fleet.Result{TaskID: t.ID, Status: "failed", Error: "receipt timeout after three deliveries"}
				continue
			}
			if !fleet.Supports(n.Status.Capabilities, t.Kind) {
				n.NeedsUpgrade = true
				continue
			}
			t.Attempts++
			t.Status = "leased"
			t.LeaseUntil = now.Add(2 * time.Minute)
			out = append(out, *clone(t))
			break
		}
		advanceRollouts(d)
		return nil
	})
	return out, err
}
func (s *Store) Result(id string, r fleet.Result, key ...string) error {
	err := s.change(func(d *diskState) error {
		if err := checkIdentity(d.Nodes[id], key); err != nil {
			return err
		}
		t := d.Tasks[r.TaskID]
		if t == nil || t.NodeID != id {
			return errors.New("unknown task")
		}
		if r.Status != "success" && r.Status != "failed" && r.Status != "unsupported" {
			return errors.New("invalid task result")
		}
		if t.Result != nil {
			return nil
		}
		if t.Status == "superseded" || t.Status == "cancelled" || t.Status == "expired" {
			return nil
		}
		if t.Status != "leased" {
			return errors.New("task not delivered")
		}
		t.Status = r.Status
		t.Result = &r
		if r.Status == "unsupported" {
			d.Nodes[id].NeedsUpgrade = true
		}
		if t.Kind == "purge" && r.Status == "success" {
			d.Nodes[id].Revoked = true
		}
		advanceRollouts(d)
		audit(d, "task-result:"+r.Status, id)
		return nil
	})
	if err == nil {
		s.notify()
	}
	return err
}

func (s *Store) ImportNode(id, name, address string) (*Node, bool, error) {
	if !fleet.SafeName(id) {
		return nil, false, errors.New("invalid imported node ID")
	}
	var out *Node
	created := false
	err := s.change(func(d *diskState) error {
		if n := d.Nodes[id]; n != nil {
			out = clone(n)
			return nil
		}
		n := &Node{ID: id, Name: name, Address: address, CreatedAt: time.Now().UTC()}
		d.Nodes[id] = n
		out = clone(n)
		created = true
		audit(d, "node-imported", id)
		return nil
	})
	return out, created, err
}

func (s *Store) WaitChannel() <-chan struct{} { s.mu.Lock(); defer s.mu.Unlock(); return s.wake }

// Backup includes credentials and signing keys and must be protected as a secret.
func (s *Store) Backup() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Marshal(struct {
		Format   int            `json:"format"`
		Identity fleet.Identity `json:"identity"`
		State    diskState      `json:"state"`
	}{1, s.Identity, s.disk})
}
func (s *Store) Restore(data []byte) error {
	var v struct {
		Format   int            `json:"format"`
		Identity fleet.Identity `json:"identity"`
		State    diskState      `json:"state"`
	}
	if len(data) > 64<<20 {
		return errors.New("backup too large")
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v.Format != 1 || v.Identity.Public() != s.Identity.Public() {
		return errors.New("backup identity mismatch; restore the original control-identity.json offline first")
	}
	if err := validateDiskState(v.State); err != nil {
		return err
	}
	if v.State.Rollouts == nil {
		v.State.Rollouts = map[string]*Rollout{}
	}
	return s.change(func(d *diskState) error {
		*d = v.State
		d.Tokens = map[string]Token{}
		audit(d, "backup-restored", "")
		return nil
	})
}
