package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
)

type Runtime interface {
	Snapshot() fleet.Status
	Apply(context.Context, fleet.DesiredState) error
	Execute(context.Context, fleet.Task) (json.RawMessage, error)
}
type Receipt struct {
	Key          string        `json:"key"`
	Kind         string        `json:"kind"`
	Payload      string        `json:"payload"`
	ExpiresAt    time.Time     `json:"expiresAt"`
	Result       *fleet.Result `json:"result,omitempty"`
	Acknowledged bool          `json:"acknowledged"`
}
type Journal struct {
	Receipts map[string]*Receipt `json:"receipts"`
}
type Agent struct {
	Dir     string
	Runtime Runtime
	Client  *Client
	mu      sync.Mutex
	journal Journal
}

func New(dir string, runtime Runtime) (*Agent, error) {
	cfg, err := LoadConfig(dir)
	if err != nil {
		return nil, err
	}
	a := &Agent{Dir: dir, Runtime: runtime, Client: &Client{Config: cfg}, journal: Journal{Receipts: map[string]*Receipt{}}}
	err = fleet.ReadJSON(filepath.Join(dir, "agent-receipts.json"), &a.journal)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if a.journal.Receipts == nil {
		return nil, errors.New("invalid agent receipts")
	}
	for id, receipt := range a.journal.Receipts {
		if id == "" || receipt == nil || receipt.Key == "" || receipt.Kind == "" || receipt.ExpiresAt.IsZero() {
			return nil, errors.New("invalid persisted agent receipt")
		}
		if receipt.Result != nil {
			if receipt.Result.TaskID != id {
				return nil, errors.New("persisted receipt task identity mismatch")
			}
			switch receipt.Result.Status {
			case "success", "failed", "unsupported", "pending-restart":
			default:
				return nil, errors.New("invalid persisted receipt status")
			}
		}
	}
	return a, nil
}
func (a *Agent) save() error {
	return fleet.WriteJSON(filepath.Join(a.Dir, "agent-receipts.json"), a.journal)
}

func (a *Agent) Process(ctx context.Context, t fleet.Task) fleet.Result {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := fleet.Result{TaskID: t.ID, Status: "failed"}
	if t.NodeID != a.Client.Config.NodeID || t.ID == "" || t.IdempotencyKey == "" {
		result.Error = "task identity mismatch"
		return result
	}
	if old := a.journal.Receipts[t.ID]; old != nil {
		if old.Key != t.IdempotencyKey || old.Kind != t.Kind || old.Payload != string(t.Payload) {
			result.Error = "task ID reused with different content"
			return result
		}
		if old.Result != nil {
			return *old.Result
		}
		if t.Kind != "apply-state" {
			result.Error = "previous execution was interrupted; inspect node before submitting a new task"
			return result
		}
	}
	if !time.Now().Before(t.ExpiresAt) {
		result.Error = "task expired"
		return result
	}
	for id, old := range a.journal.Receipts {
		if old.Key == t.IdempotencyKey && id != t.ID {
			if old.Kind != t.Kind || old.Payload != string(t.Payload) {
				result.Error = "idempotency key conflict"
				return result
			}
			if old.Result != nil {
				result = *old.Result
				result.TaskID = t.ID
				return result
			}
		}
	}
	for id, r := range a.journal.Receipts {
		if r.Acknowledged && time.Since(r.ExpiresAt) > 24*time.Hour {
			delete(a.journal.Receipts, id)
		}
	}
	if len(a.journal.Receipts) >= 10000 {
		result.Error = "receipt cache full"
		return result
	}
	a.journal.Receipts[t.ID] = &Receipt{Key: t.IdempotencyKey, Kind: t.Kind, Payload: string(t.Payload), ExpiresAt: t.ExpiresAt}
	if err := a.save(); err != nil {
		result.Error = "persist execution intent: " + err.Error()
		return result
	}
	if !fleet.Supports(a.Runtime.Snapshot().Capabilities, t.Kind) {
		result.Status = "unsupported"
		result.Error = "unsupported task kind"
	} else {
		var err error
		if t.Kind == "apply-state" {
			var s fleet.DesiredState
			if err = json.Unmarshal(t.Payload, &s); err == nil {
				err = s.Validate()
			}
			if err == nil && s.NodeID != t.NodeID {
				err = errors.New("desired-state node mismatch")
			}
			if err == nil {
				err = a.Runtime.Apply(ctx, s)
			}
		} else {
			result.Data, err = a.Runtime.Execute(ctx, t)
		}
		if errors.Is(err, fleet.ErrUpgradePending) {
			result.Status = "pending-restart"
		} else if err != nil {
			result.Error = err.Error()
		} else {
			result.Status = "success"
		}
	}
	a.journal.Receipts[t.ID].Result = &result
	if err := a.save(); err != nil {
		result.Status = "failed"
		result.Error = "persist task result: " + err.Error()
	}
	return result
}

func (a *Agent) acknowledge(ctx context.Context, r fleet.Result) error {
	var out map[string]bool
	if err := a.Client.request(ctx, "/fleet/v1/result", a.Client.Config.NodeID, r, &out); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if old := a.journal.Receipts[r.TaskID]; old != nil {
		old.Acknowledged = true
		return a.save()
	}
	return nil
}
func (a *Agent) flush(ctx context.Context) error {
	a.mu.Lock()
	var upgrade struct {
		TaskID string
		Status string
		Error  string
	}
	_ = fleet.ReadJSON(filepath.Join(a.Dir, "pending-upgrade.json"), &upgrade)
	pending := []fleet.Result{}
	for _, r := range a.journal.Receipts {
		if !r.Acknowledged && r.Result != nil {
			if r.Result.Status == "pending-restart" {
				if r.Result.TaskID == upgrade.TaskID && (upgrade.Status == "success" || upgrade.Status == "rolled-back" || upgrade.Status == "failed") {
					r.Result.Status = "success"
					if upgrade.Status != "success" {
						r.Result.Status = "failed"
						r.Result.Error = upgrade.Error
					}
					if err := a.save(); err != nil {
						a.mu.Unlock()
						return err
					}
				} else {
					a.mu.Unlock()
					return fleet.ErrUpgradePending
				}
			}
			pending = append(pending, *r.Result)
		}
	}
	a.mu.Unlock()
	for _, r := range pending {
		if err := a.acknowledge(ctx, r); err != nil {
			return err
		}
	}
	return nil
}
func (a *Agent) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		// Leave/break-glass is observed without stopping the local runtime.
		cfg, err := LoadConfig(a.Dir)
		if err != nil {
			log.Printf("fleet agent detached: %v", err)
			return
		}
		a.Client.Config = cfg
		if _, err = os.Stat(filepath.Join(a.Dir, "break-glass")); err == nil {
			return
		}
		if err = a.flush(ctx); err == nil {
			var response fleet.PollResponse
			err = a.Client.request(ctx, "/fleet/v1/poll", cfg.NodeID, fleet.Poll{Status: a.Runtime.Snapshot()}, &response)
			if err == nil {
				if len(response.Tasks) == 0 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(15 * time.Second):
					}
				}
				for _, t := range response.Tasks {
					taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
					result := a.Process(taskCtx, t)
					cancel()
					if result.Status == "pending-restart" {
						err = fleet.ErrUpgradePending
						break
					}
					if err = a.acknowledge(ctx, result); err != nil {
						break
					}
					if t.Kind == "purge" && result.Status == "success" {
						_ = os.Remove(filepath.Join(a.Dir, ConfigFile))
						return
					}
				}
			}
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("fleet connection: %v (retry in %s)", err, backoff)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		} else {
			backoff = time.Second
		}
	}
}

func (a *Agent) Report(ctx context.Context) error {
	var out map[string]bool
	err := a.Client.request(ctx, "/fleet/v1/report", a.Client.Config.NodeID, a.Runtime.Snapshot(), &out)
	if err != nil {
		return fmt.Errorf("report: %w", err)
	}
	return nil
}
