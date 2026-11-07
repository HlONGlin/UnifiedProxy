package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
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
	Encrypted    bool          `json:"encrypted,omitempty"`
	ExpiresAt    time.Time     `json:"expiresAt"`
	Result       *fleet.Result `json:"result,omitempty"`
	Acknowledged bool          `json:"acknowledged"`
}
type Journal struct {
	Receipts map[string]*Receipt `json:"receipts"`
}
type Agent struct {
	Dir       string
	Runtime   Runtime
	Client    *Client
	mu        sync.Mutex
	journal   Journal
	addressMu sync.RWMutex
	publicIP  string
	wipeOnce  sync.Once
	stopped   chan struct{}
	revoked   atomic.Bool
}

func New(dir string, runtime Runtime) (*Agent, error) {
	cfg, err := LoadConfig(dir)
	if err != nil {
		return nil, err
	}
	a := &Agent{Dir: dir, Runtime: runtime, Client: &Client{Config: cfg}, journal: Journal{Receipts: map[string]*Receipt{}}, stopped: make(chan struct{})}
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
		if old.Key != t.IdempotencyKey || old.Kind != t.Kind || old.Encrypted != t.Encrypted || old.Payload != string(t.Payload) {
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
			if old.Kind != t.Kind || old.Encrypted != t.Encrypted || old.Payload != string(t.Payload) {
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
	a.journal.Receipts[t.ID] = &Receipt{Key: t.IdempotencyKey, Kind: t.Kind, Payload: string(t.Payload), Encrypted: t.Encrypted, ExpiresAt: t.ExpiresAt}
	if err := a.save(); err != nil {
		result.Error = "persist execution intent: " + err.Error()
		return result
	}
	if !fleet.Supports(a.Runtime.Snapshot().Capabilities, t.Kind) {
		result.Status = "unsupported"
		result.Error = "unsupported task kind"
	} else {
		var err error
		runtimeTask := t
		var plaintext []byte
		if t.Encrypted {
			if !fleet.RequiresEncryptedTaskPayload(t.Kind) {
				err = errors.New("encrypted payload is not allowed for this task kind")
			} else {
				plaintext, err = fleet.DecryptTaskPayload(a.Client.Config.Encryption, t.NodeID, t.ID, t.Kind, t.Payload)
				if err == nil {
					runtimeTask.Payload = plaintext
				}
			}
		} else if fleet.RequiresEncryptedTaskPayload(t.Kind) {
			err = errors.New("task payload must be encrypted for this task kind")
		}
		if t.Kind == "apply-state" {
			if err != nil {
				result.Error = err.Error()
			} else {
				var s fleet.DesiredState
				if err = json.Unmarshal(runtimeTask.Payload, &s); err == nil {
					err = s.Validate()
				}
				if err == nil && s.NodeID != t.NodeID {
					err = errors.New("desired-state node mismatch")
				}
				if err == nil {
					err = a.Runtime.Apply(ctx, s)
				}
			}
		} else {
			if err == nil {
				result.Data, err = a.Runtime.Execute(ctx, runtimeTask)
			}
		}
		clear(plaintext)
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
	a.run(ctx, fleet.HeartbeatInterval, discoverPublicIP)
}

func (a *Agent) run(ctx context.Context, interval time.Duration, discover func(context.Context) string) {
	ctx, cancel := context.WithCancel(ctx)
	var background sync.WaitGroup
	background.Add(2)
	go func() { defer background.Done(); a.discoverAddresses(ctx, discover) }()
	go func() { defer background.Done(); a.heartbeatLoop(ctx, interval) }()
	defer func() { cancel(); background.Wait() }()
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
			err = a.Client.request(ctx, "/fleet/v1/poll", cfg.NodeID, fleet.Poll{Status: a.snapshot()}, &response)
			if err == nil {
				if len(response.Tasks) == 0 {
					if !a.wait(ctx, fleet.HeartbeatInterval) {
						return
					}
				}
				for _, t := range response.Tasks {
					taskCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
					result := a.Process(taskCtx, t)
					// Publish the actual runtime state before the panel sees completion.
					if (t.Kind == "core-control" || t.Kind == "apply-state") && result.Status == "success" {
						if reportErr := a.Report(taskCtx); reportErr != nil {
							log.Printf("fleet core status: %v", reportErr)
						}
					}
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
			if errors.Is(err, ErrAccessRevoked) {
				a.wipeRevokedAccess()
				return
			}
			if ctx.Err() != nil {
				return
			}
			log.Printf("fleet connection: %v (retry in %s)", err, backoff)
			if !a.wait(ctx, backoff) {
				return
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

func (a *Agent) wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-a.stopped:
		return false
	case <-timer.C:
		return true
	}
}

// RevokedLocally reports that this process already accepted a signed revocation
// and attempted to stop proxies and delete local data.
func (a *Agent) RevokedLocally() bool { return a.revoked.Load() }

func (a *Agent) wipeRevokedAccess() {
	a.wipeOnce.Do(func() {
		defer close(a.stopped)
		a.revoked.Store(true)
		log.Printf("fleet access revoked; stopping proxy processes and deleting node data")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if wiper, ok := a.Runtime.(interface {
			WipeProcesses(context.Context) error
		}); ok && wiper != nil {
			if err := wiper.WipeProcesses(ctx); err != nil {
				log.Printf("fleet revoke stop processes: %v", err)
			}
		}
		// Read the core path before the data directory is removed.
		if err := removeDownloadedRuntimes(a.Dir); err != nil {
			log.Printf("fleet revoke remove runtimes: %v", err)
		}
		if err := wipeManagedDir(a.Dir); err != nil {
			log.Printf("fleet revoke wipe data: %v", err)
		}
	})
}

// removeDownloadedRuntimes deletes Mihomo, Sidecar and VPNGate files that the
// node installer downloaded. Mihomo may live beside the data directory. Sidecar
// and VPNGate files live inside it. A core path outside the install directory
// is left alone.
func removeDownloadedRuntimes(dataDir string) error {
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	info, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("refusing to remove runtimes outside a real data directory")
	}
	installDir := filepath.Dir(abs)
	if installDir == abs {
		return errors.New("refusing to remove runtimes from a filesystem root")
	}
	var first error
	remember := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}
	for _, name := range []string{"caddy-naive", "shadow-tls", "ssserver"} {
		for _, suffix := range []string{"", ".exe", ".source.json"} {
			remember(removeRegularFile(filepath.Join(abs, "bin", name+suffix)))
		}
	}
	remember(removeDirInside(abs, filepath.Join(abs, "vpngate")))
	seen := map[string]bool{}
	candidates := []string{}
	if core := configuredCorePath(filepath.Join(abs, "state.json")); core != "" {
		candidates = append(candidates, core, core+".new")
	}
	for _, name := range []string{"mihomo", "mihomo.exe", "mihomo.new"} {
		candidates = append(candidates, filepath.Join(installDir, name), filepath.Join(installDir, "core", name))
	}
	for _, candidate := range candidates {
		cleaned, err := filepath.Abs(candidate)
		if err != nil {
			remember(err)
			continue
		}
		cleaned = filepath.Clean(cleaned)
		if seen[cleaned] || !mihomoFileName(cleaned) || !pathInside(installDir, cleaned) {
			seen[cleaned] = true
			continue
		}
		seen[cleaned] = true
		remember(removeRegularFile(cleaned))
	}
	return first
}

func configuredCorePath(statePath string) string {
	var state struct {
		Settings struct {
			CorePath string `json:"corePath"`
		} `json:"settings"`
	}
	if fleet.ReadJSON(statePath, &state) != nil {
		return ""
	}
	return strings.TrimSpace(state.Settings.CorePath)
}

func mihomoFileName(path string) bool {
	switch strings.ToLower(filepath.Base(path)) {
	case "mihomo", "mihomo.exe", "mihomo.new":
		return true
	default:
		return false
	}
}

func pathInside(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

func removeRegularFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("refusing to remove a directory as a runtime binary")
	}
	return os.Remove(path)
}

func removeDirInside(root, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	if !pathInside(root, abs) {
		return errors.New("refusing to remove a runtime directory outside the data directory")
	}
	info, err := os.Lstat(abs)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("refusing to remove a runtime path that is not a real directory")
	}
	return os.RemoveAll(abs)
}

// wipeManagedDir removes the contents of the node data directory. It refuses
// the filesystem root and a symlinked directory so a bad path cannot escape.
func wipeManagedDir(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	info, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("refusing to wipe a symlink or non-directory")
	}
	if filepath.Dir(abs) == abs {
		return errors.New("refusing to wipe a filesystem root")
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}
	var first error
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(abs, name)); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (a *Agent) Report(ctx context.Context) error {
	var out map[string]bool
	err := a.Client.request(ctx, "/fleet/v1/report", a.Client.Config.NodeID, a.snapshot(), &out)
	if err != nil {
		return fmt.Errorf("report: %w", err)
	}
	return nil
}
