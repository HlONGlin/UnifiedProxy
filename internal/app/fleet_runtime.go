package app

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type nodeRuntime struct {
	manager      *CoreManager
	mu           sync.Mutex
	applied      *fleet.DesiredState
	sidecars     *agent.Sidecars
	sidecarError string
}
type applyTransaction struct {
	State   State               `json:"state"`
	Config  []byte              `json:"config"`
	Applied *fleet.DesiredState `json:"applied,omitempty"`
}

func copyState(s State) State {
	b, _ := json.Marshal(s)
	var result State
	_ = json.Unmarshal(b, &result)
	return result
}
func (n *nodeRuntime) Snapshot() fleet.Status {
	n.mu.Lock()
	m := n.manager
	status := fleet.Status{Version: appVersion, Protocol: fleet.ProtocolVersion, Capabilities: append([]string(nil), fleet.TaskKinds...), OS: runtime.GOOS, Arch: runtime.GOARCH, Traffic: map[string]fleet.Usage{}}
	if n.applied != nil {
		status.Revision = n.applied.Revision
		status.Digest = n.applied.Digest
	}
	status.Error = n.sidecarError
	var sidecarStatus []agent.SidecarStatus
	if n.applied != nil && n.sidecars != nil {
		sidecarStatus = n.sidecars.Status(n.applied.Sidecars)
	}
	if runtime.GOOS != "linux" || !fleetServicePattern.MatchString(os.Getenv("UNIFIEDPROXY_SERVICE_NAME")) {
		capabilities := []string{}
		for _, kind := range status.Capabilities {
			if kind != "agent-upgrade" {
				capabilities = append(capabilities, kind)
			}
		}
		status.Capabilities = capabilities
	}
	n.mu.Unlock()
	m.mu.Lock()
	status.Running = m.runningLocked()
	started := m.started
	logs := append([]string(nil), m.logs...)
	totals := m.state.Traffic
	outboundTraffic := make(map[string]TrafficStats, len(m.state.OutboundTraffic))
	for name, usage := range m.state.OutboundTraffic {
		outboundTraffic[name] = usage
	}
	for _, in := range m.state.Inbounds {
		status.Traffic[in.ID] = fleet.Usage{LegacyCombined: in.Traffic.LegacyCombined, AllTimeLegacyCombined: in.Traffic.AllTimeLegacyCombined, Up: in.Traffic.Up, Down: in.Traffic.Down, AllTimeUp: in.Traffic.AllTimeUp, AllTimeDown: in.Traffic.AllTimeDown}
		for _, c := range in.Clients {
			status.Traffic[in.ID+":"+c.ID] = fleet.Usage{LegacyCombined: c.Traffic.LegacyCombined, AllTimeLegacyCombined: c.Traffic.AllTimeLegacyCombined, Up: c.Traffic.Up, Down: c.Traffic.Down, AllTimeUp: c.Traffic.AllTimeUp, AllTimeDown: c.Traffic.AllTimeDown, LastOnline: c.LastOnline, ExpiryTime: c.ExpiryTime, RenewCount: c.RenewCount, FirstOnline: c.FirstOnline, LastTrafficReset: c.LastTrafficReset}
		}
	}
	m.mu.Unlock()
	stats := collectSystemStats(m.dataDir)
	appendCPUSample(time.Now(), stats.CPUPercent)
	if status.Running && !started.IsZero() {
		stats.CoreUptime = uint64(time.Since(started).Seconds())
	}
	stats.PanelUptime = uint64(time.Since(m.panelStarted).Seconds())
	history := map[string][]cpuSample{}
	for _, bucket := range []int{2, 30, 60, 120, 180, 300} {
		history[fmt.Sprint(bucket)] = aggregateCPUSHistory(bucket, 60)
	}
	// Bound heartbeat details independently of configured log buffer size.
	if len(logs) > 300 {
		logs = logs[len(logs)-300:]
	}
	for i, line := range logs {
		if len(line) > 2048 {
			logs[i] = line[:2048]
		}
	}
	details := map[string]any{"system": stats, "coreVersion": m.coreVersion(), "platform": runtime.GOOS, "startedAt": started, "logs": logs, "cpuHistory": history, "totals": totals, "outboundTraffic": outboundTraffic, "vpngate": makeVPNGateView(m), "sidecars": sidecarStatus}
	if status.Running {
		if traffic, err := m.coreGet("/traffic"); err == nil {
			details["traffic"] = traffic
		}
		if connections, err := m.coreGet("/connections"); err == nil {
			applyConnectionStats(&stats, connections)
			details["system"] = stats
			if snapshot, ok := connections.(map[string]any); ok {
				if items, ok := snapshot["connections"].([]any); ok && len(items) > 200 {
					snapshot["connections"] = items[:200]
					snapshot["truncated"] = true
				}
			}
			if encoded, err := json.Marshal(connections); err == nil && len(encoded) < 512<<10 {
				details["connections"] = connections
			}
		}
	}
	status.Details, _ = json.Marshal(details)
	return status
}

func (n *nodeRuntime) recover() error {
	m := n.manager
	path := filepath.Join(m.dataDir, "apply-transaction.json")
	var tx applyTransaction
	if err := fleet.ReadJSON(path, &tx); err == nil {
		if err = n.restore(tx); err != nil {
			return err
		}
		if err = os.Remove(path); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read interrupted apply: %w", err)
	}
	var applied fleet.DesiredState
	if err := fleet.ReadJSON(filepath.Join(m.dataDir, "applied-state.json"), &applied); err == nil {
		if err = applied.Validate(); err != nil {
			return err
		}
		n.applied = &applied
		if err = n.sidecars.Apply(applied.Sidecars); err != nil {
			n.sidecarError = fmt.Sprintf("restore sidecars: %v", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (n *nodeRuntime) restore(tx applyTransaction) error {
	m := n.manager
	if err := fleet.WriteFile(filepath.Join(m.dataDir, "config.yaml"), tx.Config, 0600); err != nil {
		return err
	}
	m.mu.Lock()
	m.state = copyState(tx.State)
	err := m.saveLocked()
	m.mu.Unlock()
	if err != nil {
		return err
	}
	if tx.Applied != nil {
		if err = fleet.WriteJSON(filepath.Join(m.dataDir, "applied-state.json"), tx.Applied); err != nil {
			return err
		}
		if err = n.sidecars.Apply(tx.Applied.Sidecars); err != nil {
			return err
		}
	} else {
		if err = os.Remove(filepath.Join(m.dataDir, "applied-state.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		n.sidecars.Stop()
	}
	n.applied = tx.Applied
	return nil
}

func (n *nodeRuntime) Apply(ctx context.Context, desired fleet.DesiredState) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	m := n.manager
	if _, err := os.Stat(filepath.Join(m.dataDir, "break-glass")); err == nil {
		return errors.New("node is in break-glass mode")
	}
	if err := desired.Validate(); err != nil {
		return err
	}
	stopped, err := m.coreStopped()
	if err != nil {
		return err
	}
	if n.applied != nil && n.applied.NodeID == desired.NodeID {
		if desired.Revision < n.applied.Revision {
			return errors.New("stale desired-state revision")
		}
		if desired.Revision == n.applied.Revision {
			if desired.Digest == n.applied.Digest {
				return nil
			}
			return errors.New("revision reused with different digest")
		}
	}
	var incoming State
	if err := json.Unmarshal(desired.Model, &incoming); err != nil {
		return err
	}
	for _, in := range incoming.Inbounds {
		if err := validateInbound(in); err != nil {
			return err
		}
	}
	// Both ends use the same native renderer; this also rejects an arbitrary
	// YAML payload that does not correspond to its managed model.
	rendered, err := renderStateConfigAt(incoming, 0)
	if err != nil {
		return err
	}
	if !bytes.Equal(rendered, []byte(desired.MihomoYAML)) {
		return errors.New("desired YAML differs from the native UnifiedProxy model")
	}
	bundle := filepath.Join(m.dataDir, "fleet-configs", fmt.Sprintf("%d-%s", desired.Revision, desired.Digest[:16]))
	if err = os.MkdirAll(bundle, 0700); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, c := range desired.Certificates {
		if !fleet.SafeName(c.Name) || seen[c.Name] {
			return errors.New("invalid or duplicate certificate name")
		}
		seen[c.Name] = true
		if _, err = tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey)); err != nil {
			return fmt.Errorf("invalid certificate %s: %w", c.Name, err)
		}
		if err = fleet.WriteFile(filepath.Join(bundle, c.Name+".crt"), []byte(c.Certificate), 0600); err != nil {
			return err
		}
		if err = fleet.WriteFile(filepath.Join(bundle, c.Name+".key"), []byte(c.PrivateKey), 0600); err != nil {
			return err
		}
	}
	// Local panel credentials, API secret, executable paths and runtime counters
	// are node-owned. They never come from a controller's filesystem.
	m.mu.Lock()
	m.fleetApplying = true
	old := copyState(m.state)
	running := m.runningLocked()
	m.mu.Unlock()
	defer func() { m.mu.Lock(); m.fleetApplying = false; m.mu.Unlock() }()
	incoming.Settings = old.Settings
	if desired.NodeID != "standalone" {
		incoming.LegacyMigrations = old.LegacyMigrations
	}
	seedLegacyTraffic(&incoming, old)
	var policy State
	_ = json.Unmarshal(desired.Model, &policy)
	var previousPolicy State
	if n.applied != nil && n.applied.NodeID == desired.NodeID {
		_ = json.Unmarshal(n.applied.Model, &previousPolicy)
	}
	owned := map[string]bool{}
	for _, in := range previousPolicy.Inbounds {
		owned[in.ID] = true
	}
	for _, in := range incoming.Inbounds {
		owned[in.ID] = true
	}
	for _, in := range old.Inbounds {
		if !owned[in.ID] {
			incoming.Inbounds = append(incoming.Inbounds, in)
		}
	}
	incoming.Settings.AllowLAN = policy.Settings.AllowLAN
	incoming.Settings.Mode = policy.Settings.Mode
	incoming.Settings.LogLevel = policy.Settings.LogLevel
	for i := range incoming.Inbounds {
		in := &incoming.Inbounds[i]
		for _, prev := range old.Inbounds {
			if in.ID == prev.ID {
				in.Traffic = prev.Traffic
				carryInboundSchedule(prev, in, time.Now())
				in.Clients = mergeClientCounters(prev.Clients, in.Clients)
				for j := range in.Clients {
					client := &in.Clients[j]
					for _, previousInbound := range previousPolicy.Inbounds {
						if previousInbound.ID != in.ID {
							continue
						}
						for _, policyClient := range previousInbound.Clients {
							if policyClient.ID != client.ID || policyClient.ExpiryTime != client.ExpiryTime || policyClient.ExpireAfterDays != client.ExpireAfterDays {
								continue
							}
							for _, local := range prev.Clients {
								if local.ID == client.ID {
									client.ExpiryTime = local.ExpiryTime
								}
							}
						}
					}
				}
			}
		}
		in.Certificate = strings.ReplaceAll(in.Certificate, "__FLEET_CERT_DIR__", filepath.ToSlash(bundle))
		in.PrivateKey = strings.ReplaceAll(in.PrivateKey, "__FLEET_CERT_DIR__", filepath.ToSlash(bundle))
	}
	incoming.Traffic = old.Traffic
	incoming.OutboundTraffic = old.OutboundTraffic
	config, err := renderStateConfig(incoming)
	if err != nil {
		return err
	}
	staged := filepath.Join(bundle, "config.yaml")
	if err = fleet.WriteFile(staged, config, 0600); err != nil {
		return err
	}
	if err = n.validate(ctx, staged); err != nil {
		return err
	}
	if err = n.sidecars.Validate(ctx, desired.Sidecars); err != nil {
		return err
	}
	previous, err := m.actualConfig()
	if err != nil {
		return err
	}
	tx := applyTransaction{State: old, Config: previous, Applied: n.applied}
	transactionPath := filepath.Join(m.dataDir, "apply-transaction.json")
	if err = fleet.WriteJSON(transactionPath, tx); err != nil {
		return err
	}
	rollback := func(cause error) error {
		rollbackContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if !running {
			_ = m.stopCore()
		}
		if restoreErr := n.restore(tx); restoreErr != nil {
			return fmt.Errorf("apply failed: %v; rollback failed: %w", cause, restoreErr)
		}
		if running {
			if reloadErr := n.reload(rollbackContext, filepath.Join(m.dataDir, "config.yaml")); reloadErr != nil {
				return fmt.Errorf("apply failed: %v; rollback reload failed: %w", cause, reloadErr)
			}
		}
		if removeErr := os.Remove(transactionPath); removeErr != nil {
			return fmt.Errorf("rollback journal cleanup: %w", removeErr)
		}
		return cause
	}
	if err = fleet.WriteFile(filepath.Join(m.dataDir, "config.yaml.previous"), previous, 0600); err != nil {
		return rollback(err)
	}
	if err = fleet.WriteJSON(filepath.Join(m.dataDir, "applied-state.json.previous"), tx); err != nil {
		return rollback(err)
	}
	if err = fleet.WriteFile(filepath.Join(m.dataDir, "config.yaml"), config, 0600); err != nil {
		return rollback(err)
	}
	if running {
		if err = n.reload(ctx, filepath.Join(m.dataDir, "config.yaml")); err != nil {
			return rollback(err)
		}
	}
	if err = n.sidecars.Apply(desired.Sidecars); err != nil {
		return rollback(err)
	}
	m.mu.Lock()
	m.state = incoming
	err = m.saveLocked()
	m.initializeEnforcedClientsLocked()
	m.mu.Unlock()
	if err != nil {
		return rollback(err)
	}
	if !running && !stopped {
		if err = m.startCore(); err != nil {
			return rollback(err)
		}
	}
	if !stopped {
		if err = n.healthy(ctx); err != nil {
			return rollback(err)
		}
		if err = n.closeRestrictedSessions(ctx, old, incoming); err != nil {
			return rollback(err)
		}
	}
	if err = fleet.WriteJSON(filepath.Join(m.dataDir, "applied-state.json"), desired); err != nil {
		return rollback(err)
	}
	if err = os.Remove(transactionPath); err != nil {
		return rollback(err)
	}
	n.applied = &desired
	n.sidecarError = ""
	go m.syncVPNGateTasks()
	return nil
}

func (n *nodeRuntime) supervise(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if _, err := os.Stat(filepath.Join(n.manager.dataDir, "break-glass")); err == nil {
			continue
		}
		n.mu.Lock()
		if n.applied != nil {
			job, cancel := context.WithTimeout(ctx, 2*time.Minute)
			err := n.sidecars.Provision(job, n.applied.Sidecars)
			cancel()
			if err == nil {
				err = n.sidecars.Apply(n.applied.Sidecars)
			}
			if err != nil {
				n.sidecarError = err.Error()
			} else {
				n.sidecarError = ""
			}
		}
		n.mu.Unlock()
	}
}

func (n *nodeRuntime) validate(ctx context.Context, path string) error {
	m := n.manager
	settings := m.settingsSnapshot()
	cmd := exec.CommandContext(ctx, settings.CorePath, "-d", m.dataDir, "-t", "-f", path)
	cmd.Dir = m.dataDir
	// Core output may include credentials, so report only exit status.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Mihomo rejected staged configuration: %w", err)
	}
	return nil
}
func (n *nodeRuntime) coreHTTP(ctx context.Context, method, path string, body []byte) error {
	settings := n.manager.settingsSnapshot()
	host, _, err := net.SplitHostPort(settings.APIAddress)
	if err != nil {
		return errors.New("invalid local core API address")
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("fleet runtime requires a loopback core API")
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://"+settings.APIAddress+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+settings.APISecret)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Mihomo API returned HTTP %d", response.StatusCode)
	}
	return nil
}
func (n *nodeRuntime) reload(ctx context.Context, path string) error {
	b, _ := json.Marshal(map[string]string{"path": path})
	return n.coreHTTP(ctx, "PUT", "/configs?force=true", b)
}
func (n *nodeRuntime) healthy(ctx context.Context) error {
	for i := 0; i < 20; i++ {
		if err := n.coreHTTP(ctx, "GET", "/version", nil); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return errors.New("Mihomo health check failed")
}
func (n *nodeRuntime) Execute(ctx context.Context, t fleet.Task) (json.RawMessage, error) {
	if t.Kind != "collect-status" && t.Kind != "test-egress" && t.Kind != "outbound-delay" {
		if _, err := os.Stat(filepath.Join(n.manager.dataDir, "break-glass")); err == nil {
			return nil, errors.New("node is in break-glass mode")
		}
	}
	switch t.Kind {
	case "core-control":
		var p struct {
			Action string `json:"action"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, err
		}
		switch p.Action {
		case "start", "stop", "restart":
			n.mu.Lock()
			defer n.mu.Unlock()
			if err := n.manager.controlCore(p.Action); err != nil {
				return nil, err
			}
			if p.Action != "stop" {
				if err := n.healthy(ctx); err != nil {
					return nil, err
				}
			}
			n.manager.mu.Lock()
			running := n.manager.runningLocked()
			n.manager.mu.Unlock()
			return json.Marshal(map[string]bool{"running": running})
		case "validate":
			return nil, n.validate(ctx, filepath.Join(n.manager.dataDir, "config.yaml"))
		case "close-connections":
			path := "/connections"
			if p.ID != "" {
				path += "/" + url.PathEscape(p.ID)
			}
			return nil, n.coreHTTP(ctx, "DELETE", path, nil)
		}
		return nil, errors.New("unknown core action")
	case "traffic-reset":
		r, _ := http.NewRequestWithContext(ctx, "POST", "/api/traffic/reset", bytes.NewReader(t.Payload))
		w := &captureResponse{header: make(http.Header)}
		(&App{manager: n.manager}).handleTrafficReset(w, r)
		if w.code >= 300 {
			return nil, errors.New("traffic reset failed")
		}
		if err := n.manager.writeConfig(); err != nil {
			return nil, err
		}
		if err := n.reload(ctx, filepath.Join(n.manager.dataDir, "config.yaml")); err != nil {
			return nil, err
		}
		return json.RawMessage(w.body.Bytes()), nil
	case "vpngate-install":
		if n.manager.vpnGateInstalled() {
			return json.Marshal(makeVPNGateView(n.manager))
		}
		if err := vpnGatePreflight(); err != nil {
			return nil, err
		}
		if err := n.manager.runVPNGateInstall(ctx); err != nil {
			return nil, err
		}
		n.manager.syncVPNGateTasks()
		return json.Marshal(makeVPNGateView(n.manager))
	case "vpngate-reconnect":
		r, _ := http.NewRequestWithContext(ctx, "POST", "/api/mihomo/vpngate/reconnect", bytes.NewReader(t.Payload))
		w := &captureResponse{header: make(http.Header)}
		(&App{manager: n.manager}).reconnectVPNGateSlot(w, r)
		if w.code >= 300 {
			return nil, errors.New("VPNGate reconnect failed")
		}
		return json.RawMessage(w.body.Bytes()), nil
	case "collect-status":
		return json.Marshal(n.Snapshot())
	case "test-egress":
		return n.testEgress(ctx, t.Payload)
	case "outbound-delay":
		r, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/api/mihomo/outbound/delay", bytes.NewReader(t.Payload))
		w := &captureResponse{header: make(http.Header)}
		(&App{manager: n.manager}).handleMihomoOutboundDelay(w, r)
		var result struct {
			Data    json.RawMessage `json:"data"`
			Message string          `json:"message"`
		}
		if err := json.Unmarshal(w.body.Bytes(), &result); err != nil {
			return nil, err
		}
		if w.code >= 300 {
			return nil, errors.New(result.Message)
		}
		return result.Data, nil
	case "kernel-switch":
		n.mu.Lock()
		defer n.mu.Unlock()
		var p struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, err
		}
		v, err := n.manager.installCore(ctx, p.Version)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]string{"version": v})
	case "geo-update":
		n.mu.Lock()
		defer n.mu.Unlock()
		return nil, n.manager.updateGeofiles(ctx)
	case "cert-renew":
		// Certificate acquisition runs centrally; this task revalidates the
		// distributed pair and reloads it without opening an ACME client on nodes.
		n.mu.Lock()
		defer n.mu.Unlock()
		if n.applied == nil {
			return nil, errors.New("no managed certificates")
		}
		for _, c := range n.applied.Certificates {
			if _, err := tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey)); err != nil {
				return nil, err
			}
		}
		return nil, n.reload(ctx, filepath.Join(n.manager.dataDir, "config.yaml"))
	case "agent-upgrade":
		return nil, n.upgrade(ctx, t)
	case "purge":
		n.sidecars.Stop()
		n.manager.stopVPNGateTasks()
		return nil, n.manager.stopCore()
	default:
		return nil, fmt.Errorf("unsupported task %s", t.Kind)
	}
}

func (n *nodeRuntime) testEgress(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Proxy string `json:"proxy"`
		URL   string `json:"url"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return nil, err
	}
	if p.Proxy == "" {
		return nil, errors.New("a node-local SOCKS5 proxy is required")
	}
	return testFleetEgress(ctx, p.Proxy, p.URL)
}

// The updater verifies a controller-provided SHA256 before touching the executable.
func (n *nodeRuntime) upgrade(ctx context.Context, task fleet.Task) error {
	var p struct {
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(task.Payload, &p); err != nil {
		return err
	}
	if err := stageFleetUpgrade(ctx, n.manager.dataDir, p.URL, p.SHA256); err != nil {
		return err
	}
	if err := startFleetUpgrade(n.manager.dataDir, n.manager.settingsSnapshot(), task.ID); err != nil {
		return err
	}
	return fleet.ErrUpgradePending
}
