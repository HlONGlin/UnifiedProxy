package app

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type fleetUpgrade struct {
	Executable string    `json:"executable"`
	SHA256     string    `json:"sha256"`
	Service    string    `json:"service"`
	HealthURL  string    `json:"healthURL"`
	HealthRoot string    `json:"healthRoot,omitempty"`
	TLSName    string    `json:"tlsName,omitempty"`
	TaskID     string    `json:"taskId,omitempty"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
	WorkerUnit string    `json:"workerUnit,omitempty"`
	StartedAt  time.Time `json:"startedAt,omitempty"`
	Phase      string    `json:"phase,omitempty"`
}

var fleetServicePattern = regexp.MustCompile(`^unifiedproxy-(control|node)[a-zA-Z0-9_.-]*[.]service$`)

func executableDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func startFleetUpgrade(dir string, settings Settings, taskID string) error {
	if runtime.GOOS != "linux" {
		return errors.New("automatic upgrade requires Linux systemd")
	}
	service := os.Getenv("UNIFIEDPROXY_SERVICE_NAME")
	if !fleetServicePattern.MatchString(service) {
		return errors.New("UNIFIEDPROXY_SERVICE_NAME must name an installed UnifiedProxy control/node systemd service")
	}
	host, port, err := net.SplitHostPort(settings.PanelListen)
	if err != nil {
		return err
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("upgrade health listener must be loopback or wildcard")
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "pending-upgrade.json")
	var pending fleetUpgrade
	if err = fleet.ReadJSON(path, &pending); err != nil {
		return err
	}
	if pending.Status != "staged" {
		return errors.New("no staged upgrade")
	}
	pending.Service = service
	pending.TaskID = taskID
	pending.HealthURL = "http://" + net.JoinHostPort(host, port) + "/fleet/v1/health"
	if settings.PanelCertFile != "" || settings.PanelKeyFile != "" {
		certificate, err := os.ReadFile(settings.PanelCertFile)
		if err != nil {
			return err
		}
		pair, err := tls.LoadX509KeyPair(settings.PanelCertFile, settings.PanelKeyFile)
		if err != nil {
			return err
		}
		leaf, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			return err
		}
		name := settings.PanelDomain
		if name == "" && len(leaf.DNSNames) > 0 {
			name = strings.Replace(leaf.DNSNames[0], "*.", "probe.", 1)
		}
		if name == "" {
			name = host
		}
		pending.HealthRoot = string(certificate)
		pending.TLSName = name
		pending.HealthURL = "https://" + net.JoinHostPort(host, port) + "/fleet/v1/health"
	}
	pending.Status = "pending-restart"
	pending.WorkerUnit = "unifiedproxy-upgrade-" + fleet.ID()[:16] + ".service"
	pending.StartedAt = time.Now().UTC()
	pending.Phase = "scheduled"
	if err = fleet.WriteJSON(path, pending); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// A transient unit survives stopping the application's own control group.
	// Do not wait for the oneshot job: it stops that group during the update.
	cmd := exec.CommandContext(ctx, "systemd-run", "--quiet", "--collect", "--no-block", "--unit="+pending.WorkerUnit, "--property=Type=oneshot", pending.Executable, "upgrade-worker", "--data-dir", dir)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err = cmd.Run(); err != nil {
		pending.Status = "failed"
		pending.Error = "cannot schedule systemd update worker"
		_ = fleet.WriteJSON(path, pending)
		return err
	}
	return nil
}

func runFleetUpgradeWorker(ctx context.Context, dir string) (resultErr error) {
	path := filepath.Join(dir, "pending-upgrade.json")
	var p fleetUpgrade
	if err := fleet.ReadJSON(path, &p); err != nil {
		return err
	}
	defer func() {
		if resultErr != nil {
			var latest fleetUpgrade
			if fleet.ReadJSON(path, &latest) == nil && latest.Status == "pending-restart" {
				latest.Status = "failed"
				latest.Error = resultErr.Error()
				_ = fleet.WriteJSON(path, latest)
			}
		}
	}()
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if runtime.GOOS != "linux" || !fleetServicePattern.MatchString(p.Service) || filepath.Clean(self) != filepath.Clean(p.Executable) || p.Status != "pending-restart" {
		return errors.New("invalid update worker context")
	}
	u, err := http.NewRequestWithContext(ctx, "GET", p.HealthURL, nil)
	if err != nil {
		return err
	}
	host := net.ParseIP(u.URL.Hostname())
	if (u.URL.Scheme != "http" && u.URL.Scheme != "https") || (u.URL.Hostname() != "localhost" && (host == nil || !host.IsLoopback())) {
		return errors.New("invalid health probe")
	}
	digest, err := executableDigest(p.Executable + ".next")
	if err != nil || digest != p.SHA256 {
		return errors.New("candidate checksum changed")
	}
	// Allow the old agent to flush its pending receipt before stopping it.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}
	service := func(action string) error { return fleetSystemService(action, p.Service) }
	save := func(status string, cause error) error {
		p.Status = status
		if cause != nil {
			p.Error = cause.Error()
		}
		return fleet.WriteJSON(path, p)
	}
	// Copy the rollback binary before stopping; Linux can atomically replace a
	// running executable. There is no boot-time gap with a missing ExecStart.
	previous := p.Executable + ".previous"
	oldBinary, err := os.ReadFile(p.Executable)
	if err != nil {
		return err
	}
	if err = fleet.WriteFile(previous, oldBinary, 0755); err != nil {
		return err
	}
	p.Phase = "backed-up"
	if err = fleet.WriteJSON(path, p); err != nil {
		return err
	}
	if err = service("stop"); err != nil {
		_ = save("failed", err)
		return err
	}
	rollback := func(cause error) error {
		_ = service("stop")
		oldBinary, readErr := os.ReadFile(previous)
		if readErr != nil {
			_ = save("failed", readErr)
			return readErr
		}
		if err := fleet.WriteFile(p.Executable, oldBinary, 0755); err != nil {
			_ = save("failed", fmt.Errorf("update: %v; restore: %w", cause, err))
			return err
		}
		if err := service("start"); err != nil {
			_ = save("failed", err)
			return err
		}
		_ = save("rolled-back", cause)
		return cause
	}
	if err = os.Rename(p.Executable+".next", p.Executable); err != nil {
		return rollback(err)
	}
	p.Phase = "verifying"
	if err = fleet.WriteJSON(path, p); err != nil {
		return rollback(err)
	}
	if err = service("start"); err != nil {
		return rollback(err)
	}
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if u.URL.Scheme == "https" {
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM([]byte(p.HealthRoot)) {
			return rollback(errors.New("invalid health TLS trust"))
		}
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: p.TLSName, MinVersion: tls.VersionTLS12}}
	}
	for i := 0; i < 45; i++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", p.HealthURL, nil)
		res, requestErr := client.Do(req)
		if requestErr == nil {
			var health struct{ SHA256 string }
			decodeErr := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&health)
			_ = res.Body.Close()
			if res.StatusCode == 200 && decodeErr == nil && strings.EqualFold(health.SHA256, p.SHA256) {
				return save("success", nil)
			}
		}
		select {
		case <-ctx.Done():
			return rollback(ctx.Err())
		case <-time.After(time.Second):
		}
	}
	return rollback(errors.New("candidate health verification timed out"))
}

func fleetSystemService(action, name string) error {
	job, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(job, "systemctl", action, name)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	return cmd.Run()
}

// A persistent timer runs this command from the installer's recovery binary.
// It handles worker termination or a reboot without acknowledging an
// unverified candidate. The normal worker has five minutes to finish.
func recoverFleetUpgrade(dir string) error {
	path := filepath.Join(dir, "pending-upgrade.json")
	var p fleetUpgrade
	if err := fleet.ReadJSON(path, &p); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if p.Status != "pending-restart" || (!p.StartedAt.IsZero() && time.Since(p.StartedAt) < 10*time.Minute) {
		return nil
	}
	if runtime.GOOS != "linux" || !fleetServicePattern.MatchString(p.Service) || !filepath.IsAbs(p.Executable) {
		return errors.New("invalid upgrade recovery context")
	}
	if regexp.MustCompile(`^unifiedproxy-upgrade-[a-f0-9]{16}[.]service$`).MatchString(p.WorkerUnit) && fleetSystemService("is-active", p.WorkerUnit) == nil {
		return nil
	}
	if p.Phase == "backed-up" || p.Phase == "verifying" {
		previous, err := os.ReadFile(p.Executable + ".previous")
		if err != nil {
			return fmt.Errorf("read recovery binary: %w", err)
		}
		if err = fleetSystemService("stop", p.Service); err != nil {
			return err
		}
		if err = fleet.WriteFile(p.Executable, previous, 0755); err != nil {
			_ = fleetSystemService("start", p.Service)
			return err
		}
		if err = fleetSystemService("start", p.Service); err != nil {
			return err
		}
		p.Status = "rolled-back"
	} else {
		p.Status = "failed"
	}
	p.Error = "upgrade worker interrupted; recovery timer restored the previous binary when necessary"
	return fleet.WriteJSON(path, p)
}

func (a *App) handleFleetUpdate(w http.ResponseWriter, r *http.Request) {
	a.fleet.mu.Lock()
	defer a.fleet.mu.Unlock()
	if r.Method == "GET" {
		var pending fleetUpgrade
		_ = fleet.ReadJSON(filepath.Join(a.manager.dataDir, "pending-upgrade.json"), &pending)
		writeJSON(w, 200, apiResponse{OK: true, Data: pending})
		return
	}
	if r.Method != "POST" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	var p struct {
		URL    string
		SHA256 string
	}
	if err := decodeJSON(r, &p); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	err := stageFleetUpgrade(r.Context(), a.manager.dataDir, p.URL, p.SHA256)
	if err == nil {
		err = startFleetUpgrade(a.manager.dataDir, a.manager.settingsSnapshot(), "")
	}
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	writeJSON(w, 202, apiResponse{OK: true, Message: "verified update scheduled; health failure restores the previous binary"})
}
