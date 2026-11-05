package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type process struct {
	cmd    *exec.Cmd
	digest string
	done   chan struct{}
}
type Sidecars struct {
	Dir       string
	mu        sync.Mutex
	processes map[string]*process
}
type ShadowConfig struct {
	Listen          string `json:"listen"`
	Backend         string `json:"backend"`
	Handshake       string `json:"handshake"`
	Password        string `json:"password"`
	ManagedBackend  bool   `json:"managedBackend"`
	BackendPassword string `json:"backendPassword"`
	Cipher          string `json:"cipher"`
}

type SidecarStatus struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Enabled        bool   `json:"enabled"`
	Running        bool   `json:"running"`
	Release        string `json:"release"`
	RecordedSHA256 string `json:"recordedSHA256,omitempty"`
}

func (s *Sidecars) Status(configs []fleet.Sidecar) []SidecarStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := []SidecarStatus{}
	for _, config := range configs {
		status := SidecarStatus{ID: config.ID, Kind: config.Kind, Enabled: config.Enabled, Release: "locally-provisioned"}
		if p := s.processes[config.ID]; p != nil {
			select {
			case <-p.done:
			default:
				status.Running = true
			}
		}
		name := map[string]string{"naive": "caddy-naive", "shadowtls": "shadow-tls", "shadowsocks": "ssserver"}[config.Kind]
		var provenance struct {
			Source string
			SHA256 string
		}
		if name != "" && fleet.ReadJSON(s.binary(name)+".source.json", &provenance) == nil {
			status.Release = provenance.Source
			status.RecordedSHA256 = provenance.SHA256
		}
		result = append(result, status)
	}
	return result
}

func (s *Sidecars) binary(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(s.Dir, "bin", name)
}

// Provision restores missing, fixed-version executables without validating or
// rewriting configurations of healthy processes on each supervision tick.
func (s *Sidecars) Provision(ctx context.Context, configs []fleet.Sidecar) error {
	for _, config := range configs {
		if config.Enabled {
			if err := s.ensure(ctx, config.Kind); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *Sidecars) specification(c fleet.Sidecar) (string, []string, error) {
	if !fleet.SafeName(c.ID) {
		return "", nil, errors.New("invalid sidecar ID")
	}
	p := filepath.Join(s.Dir, "sidecars", c.ID+".json")
	switch c.Kind {
	case "naive":
		var cfg map[string]any
		if err := json.Unmarshal([]byte(c.Config), &cfg); err != nil {
			return "", nil, err
		}
		return s.binary("caddy-naive"), []string{"run", "--config", p}, nil
	case "shadowtls":
		var cfg ShadowConfig
		if err := json.Unmarshal([]byte(c.Config), &cfg); err != nil {
			return "", nil, err
		}
		if cfg.Listen == "" || cfg.Backend == "" || cfg.Handshake == "" || cfg.Password == "" {
			return "", nil, errors.New("incomplete ShadowTLS configuration")
		}
		return s.binary("shadow-tls"), []string{"--v3", "server", "--listen", cfg.Listen, "--server", cfg.Backend, "--tls", cfg.Handshake, "--password", cfg.Password}, nil
	case "shadowsocks":
		return s.binary("ssserver"), []string{"-c", p}, nil
	default:
		return "", nil, fmt.Errorf("unsupported sidecar %q", c.Kind)
	}
}

// Validate never disrupts the running processes.
func (s *Sidecars) Validate(ctx context.Context, configs []fleet.Sidecar) error {
	seen := map[string]bool{}
	for _, c := range configs {
		if seen[c.ID] {
			return errors.New("duplicate sidecar ID")
		}
		seen[c.ID] = true
		if !c.Enabled {
			continue
		}
		if err := s.ensure(ctx, c.Kind); err != nil {
			return err
		}
		bin, _, err := s.specification(c)
		if err != nil {
			return err
		}
		if _, err = os.Stat(bin); err != nil {
			return fmt.Errorf("sidecar executable is missing: %s", bin)
		}
		if !json.Valid([]byte(c.Config)) {
			return errors.New("sidecar configuration must be JSON")
		}
		if c.Kind == "naive" {
			p, err := os.CreateTemp(s.Dir, ".caddy-check-*.json")
			if err != nil {
				return err
			}
			path := p.Name()
			_, err = p.WriteString(c.Config)
			_ = p.Close()
			if err != nil {
				os.Remove(path)
				return err
			}
			cmd := exec.CommandContext(ctx, bin, "validate", "--config", path)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			err = cmd.Run()
			os.Remove(path)
			if err != nil {
				return fmt.Errorf("Caddy validation failed: %w", err)
			}
		}
	}
	return nil
}
func stopProcess(p *process) {
	if p == nil {
		return
	}
	_ = p.cmd.Process.Kill()
	select {
	case <-p.done:
	case <-time.After(5 * time.Second):
	}
}
func (s *Sidecars) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, p := range s.processes {
		stopProcess(p)
		delete(s.processes, id)
	}
}
func (s *Sidecars) Apply(configs []fleet.Sidecar) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processes == nil {
		s.processes = map[string]*process{}
	}
	desired := map[string]bool{}
	for _, c := range configs {
		if !c.Enabled {
			continue
		}
		desired[c.ID] = true
		bin, args, err := s.specification(c)
		if err != nil {
			return err
		}
		h := sha256.Sum256([]byte(c.Kind + "\n" + c.Config))
		digest := hex.EncodeToString(h[:])
		if p := s.processes[c.ID]; p != nil && p.digest == digest {
			select {
			case <-p.done:
			default:
				continue
			}
		}
		if err = fleet.WriteFile(filepath.Join(s.Dir, "sidecars", c.ID+".json"), []byte(c.Config), 0600); err != nil {
			return err
		}
		stopProcess(s.processes[c.ID])
		delete(s.processes, c.ID)
		cmd := exec.Command(bin, args...)
		cmd.Dir = s.Dir
		cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+filepath.Join(s.Dir, "sidecars", c.ID, "config"), "XDG_DATA_HOME="+filepath.Join(s.Dir, "sidecars", c.ID, "data"), "MONOIO_FORCE_LEGACY_DRIVER=1")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err = cmd.Start(); err != nil {
			return fmt.Errorf("start sidecar %s: %w", c.ID, err)
		}
		p := &process{cmd: cmd, digest: digest, done: make(chan struct{})}
		s.processes[c.ID] = p
		go func() { _ = cmd.Wait(); close(p.done) }()
		select {
		case <-p.done:
			return fmt.Errorf("sidecar %s exited during startup", c.ID)
		case <-time.After(150 * time.Millisecond):
		}
	}
	for id, p := range s.processes {
		if !desired[id] {
			stopProcess(p)
			delete(s.processes, id)
		}
	}
	return nil
}
