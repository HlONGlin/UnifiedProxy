package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/control"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
	installers "github.com/hlonglin/unifiedproxy/scripts"
)

type fleetConsole struct {
	store   *control.Store
	server  *control.Server
	dataDir string
	mu      sync.Mutex
	editors map[string]*App
}

func (a *App) withFleetTransport(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fleet/v1/install.sh" {
			if a.fleet == nil {
				http.NotFound(w, r)
				return
			}
			if r.Method != "GET" && r.Method != "HEAD" {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "GET required", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if r.Method == "GET" {
				_, _ = io.WriteString(w, installers.NodeInstaller)
			}
			return
		}
		if r.URL.Path == "/fleet/v1/health" && r.Method == "GET" {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			ip := net.ParseIP(host)
			if ip == nil || !ip.IsLoopback() {
				http.NotFound(w, r)
				return
			}
			if a.nodeRuntime != nil {
				a.nodeRuntime.mu.Lock()
				sidecarError := a.nodeRuntime.sidecarError
				a.nodeRuntime.mu.Unlock()
				a.manager.mu.Lock()
				running := a.manager.runningLocked()
				a.manager.mu.Unlock()
				if !running || sidecarError != "" {
					http.Error(w, "runtime unhealthy", http.StatusServiceUnavailable)
					return
				}
			}
			path, err := os.Executable()
			digest := ""
			if err == nil {
				digest, err = executableDigest(path)
			}
			if err != nil {
				http.Error(w, "unhealthy", 503)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"version": appVersion, "sha256": digest})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/fleet/v1/") {
			if a.fleet == nil {
				http.NotFound(w, r)
				return
			}
			a.fleet.server.ServeHTTP(w, r)
			return
		}
		if (a.fleet != nil || a.nodeRuntime != nil) && strings.HasPrefix(r.URL.Path, "/_UnifiedProxy/peer/") {
			http.Error(w, "P2P is disabled in fleet mode", http.StatusGone)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) withFleetPanel(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.nodeRuntime != nil && strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/api/fleet") {
			_, attached := os.Stat(filepath.Join(a.manager.dataDir, "agent.json"))
			_, glass := os.Stat(filepath.Join(a.manager.dataDir, "break-glass"))
			if attached == nil && glass != nil && r.Method != "GET" && r.Method != "HEAD" && !strings.HasPrefix(r.URL.Path, "/api/auth/") {
				a.nodeRuntime.mu.Lock()
				defer a.nodeRuntime.mu.Unlock()
			}
			if attached == nil && glass != nil && r.Method != "GET" && r.Method != "HEAD" && !strings.HasPrefix(r.URL.Path, "/api/auth/") && !a.localUnmanagedMutation(r) {
				writeJSON(w, 409, apiResponse{Message: "managed node is read-only; use agent break-glass for emergency takeover"})
				return
			}
		}
		if a.fleet != nil && strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/api/fleet") {
			if r.URL.Path == "/api/multi-control" || strings.HasPrefix(r.URL.Path, "/api/multi-control/") || r.URL.Path == "/api/inbound-sync" {
				writeJSON(w, 410, apiResponse{Message: "P2P synchronization is replaced by fleet management"})
				return
			}
			if id := r.Header.Get("X-Fleet-Node"); id != "" {
				if a.remoteRoute(r.URL.Path) {
					a.auth(func(w http.ResponseWriter, r *http.Request) { a.handleRemotePanel(w, r, id) })(w, r)
					return
				}
			}
			if r.Method != "GET" && (r.URL.Path == "/api/inbounds" || r.URL.Path == "/api/clients" || strings.HasPrefix(r.URL.Path, "/api/core/") || r.URL.Path == "/api/geofiles/update") {
				writeJSON(w, 409, apiResponse{Message: "select a fleet node before changing runtime objects"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) remoteRoute(path string) bool {
	switch path {
	case "/api/state", "/api/config", "/api/backup", "/api/logs/download", "/api/subscriptions/token", "/api/inbounds", "/api/inbounds/import", "/api/clients", "/api/raw-config", "/api/config/download", "/api/traffic/reset", "/api/mihomo/settings", "/api/mihomo/outbound/parse", "/api/mihomo/warp", "/api/mihomo/vpngate", "/api/core/test", "/api/core/install", "/api/geofiles/update", "/api/core/logs", "/api/import/xray":
		return true
	case "/api/tools/uuid", "/api/tools/inbound-port", "/api/tools/subscription-path", "/api/tools/reality-keypair", "/api/tools/vless-encryption", "/api/tools/ech-keypair":
		return true
	case "/api/core/start", "/api/core/stop", "/api/core/restart", "/api/core/connections":
		return true
	case "/api/mihomo/outbound/delay":
		return true
	}
	return strings.HasPrefix(path, "/api/mihomo/vpngate/") || strings.HasPrefix(path, "/api/mihomo/warp/") || strings.HasPrefix(path, "/api/server/cpuHistory/") || strings.HasPrefix(path, "/api/node-tasks/")
}

func (f *fleetConsole) editor(id string) (*App, error) {
	if !fleet.SafeName(id) {
		return nil, errors.New("invalid node ID")
	}
	n := f.store.Node(id)
	if n == nil || n.Revoked {
		return nil, errors.New("unknown or revoked node")
	}
	if app := f.editors[id]; app != nil {
		return app, nil
	}
	dir := filepath.Join(f.dataDir, "fleet-nodes", id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	m := &CoreManager{remote: true, dataDir: dir, panelStarted: time.Now(), connectionStats: map[string]connectionSample{}, enforcedClients: map[string]bool{}}
	_, draftErr := os.Stat(filepath.Join(dir, "state.json"))
	if draftErr != nil && !errors.Is(draftErr, os.ErrNotExist) {
		return nil, draftErr
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	if errors.Is(draftErr, os.ErrNotExist) && n.Desired != nil {
		if err := json.Unmarshal(n.Desired.Model, &m.state); err != nil {
			return nil, err
		}
		for i := range m.state.Inbounds {
			in := &m.state.Inbounds[i]
			for _, c := range n.Desired.Certificates {
				if in.Certificate == "__FLEET_CERT_DIR__/"+c.Name+".crt" {
					in.Certificate = c.Certificate
					in.PrivateKey = c.PrivateKey
					in.CertificateMode = "content"
				}
			}
		}
		if err := m.saveLocked(); err != nil {
			return nil, err
		}
	}
	app := &App{manager: m, role: "editor"}
	f.editors[id] = app
	return app, nil
}

func (f *fleetConsole) publish(id string, app *App, sidecars []fleet.Sidecar) (*fleet.DesiredState, error) {
	m := app.manager
	m.mu.Lock()
	state := copyState(m.state)
	m.mu.Unlock()
	state.Settings.Username = ""
	state.Settings.Password = ""
	state.Settings.CorePath = ""
	state.Settings.PanelCertFile = ""
	state.Settings.PanelKeyFile = ""
	// Policy digests exclude timestamps and node-owned runtime counters.
	state.UpdatedAt = ""
	state.Traffic = TrafficTotals{}
	state.OutboundTraffic = nil
	state.SubscriptionTokens = nil
	state.LegacyMigrations = nil
	for i := range state.Inbounds {
		in := &state.Inbounds[i]
		in.Traffic = TrafficStats{}
		in.LastTrafficReset = ""
		in.SyncOrigin = nil
		for j := range in.Clients {
			c := &in.Clients[j]
			c.Traffic = TrafficStats{}
			c.LastOnline = ""
			c.FirstOnline = ""
			c.LastTrafficReset = ""
			c.RenewCount = 0
		}
	}
	certificates := []fleet.Certificate{}
	for i := range state.Inbounds {
		in := &state.Inbounds[i]
		if !inboundLoadsCertificate(*in) || in.Certificate == "" || in.PrivateKey == "" {
			continue
		}
		cert, key := []byte(in.Certificate), []byte(in.PrivateKey)
		if !strings.Contains(in.Certificate, "-----BEGIN") {
			var err error
			cert, err = os.ReadFile(in.Certificate)
			if err != nil {
				return nil, fmt.Errorf("read certificate for %s: %w", in.Name, err)
			}
		}
		if !strings.Contains(in.PrivateKey, "-----BEGIN") {
			var err error
			key, err = os.ReadFile(in.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("read private key for %s: %w", in.Name, err)
			}
		}
		name := "inbound-" + in.ID
		if !fleet.SafeName(name) {
			return nil, errors.New("invalid inbound ID for certificate")
		}
		certificates = append(certificates, fleet.Certificate{Name: name, Certificate: string(cert), PrivateKey: string(key)})
		in.Certificate = "__FLEET_CERT_DIR__/" + name + ".crt"
		in.PrivateKey = "__FLEET_CERT_DIR__/" + name + ".key"
		in.CertificateMode = "file"
	}
	// Wire YAML is a stable policy projection. Local clocks enforce expiry.
	yaml, err := renderStateConfigAt(state, 0)
	if err != nil {
		return nil, err
	}
	model, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	if sidecars == nil {
		if n := f.store.Node(id); n != nil && n.Desired != nil {
			sidecars = n.Desired.Sidecars
		}
	}
	return f.store.SetDesired(id, fleet.DesiredState{MihomoYAML: string(yaml), Model: model, Certificates: certificates, Sidecars: sidecars})
}

type captureResponse struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func (c *captureResponse) Header() http.Header { return c.header }
func (c *captureResponse) WriteHeader(code int) {
	if c.code == 0 {
		c.code = code
	}
}
func (c *captureResponse) Write(b []byte) (int, error) {
	if c.code == 0 {
		c.code = 200
	}
	return c.body.Write(b)
}

func (a *App) handleRemotePanel(w http.ResponseWriter, r *http.Request, id string) {
	f := a.fleet
	f.mu.Lock()
	defer f.mu.Unlock()
	editor, err := f.editor(id)
	if err != nil {
		writeJSON(w, 404, apiResponse{Message: err.Error()})
		return
	}
	if r.URL.Path == "/api/core/install" || r.URL.Path == "/api/geofiles/update" || r.URL.Path == "/api/core/test" || r.URL.Path == "/api/core/start" || r.URL.Path == "/api/core/stop" || r.URL.Path == "/api/core/restart" || r.URL.Path == "/api/core/connections" || r.URL.Path == "/api/traffic/reset" || r.URL.Path == "/api/mihomo/vpngate/install" || r.URL.Path == "/api/mihomo/vpngate/reconnect" {
		if r.Method != "POST" && !(r.URL.Path == "/api/core/connections" && r.Method == "DELETE") {
			writeJSON(w, 405, apiResponse{})
			return
		}
		kind := "collect-status"
		payload := json.RawMessage(`{}`)
		if r.URL.Path == "/api/core/install" {
			kind = "kernel-switch"
			var p struct {
				Tag string `json:"tag"`
			}
			if err := decodeJSON(r, &p); err != nil {
				writeJSON(w, 400, apiResponse{Message: err.Error()})
				return
			}
			payload, _ = json.Marshal(map[string]string{"version": p.Tag})
		}
		if r.URL.Path == "/api/geofiles/update" {
			kind = "geo-update"
		}
		if strings.HasPrefix(r.URL.Path, "/api/core/") && r.URL.Path != "/api/core/install" {
			kind = "core-control"
			action := strings.TrimPrefix(r.URL.Path, "/api/core/")
			if action == "test" {
				action = "validate"
			}
			if action == "connections" {
				action = "close-connections"
			}
			payload, _ = json.Marshal(map[string]string{"action": action, "id": r.URL.Query().Get("id")})
		}
		if r.URL.Path == "/api/traffic/reset" || r.URL.Path == "/api/mihomo/vpngate/reconnect" {
			data, readErr := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if readErr != nil || !json.Valid(data) {
				writeJSON(w, 400, apiResponse{Message: "invalid task payload"})
				return
			}
			payload = data
			kind = "traffic-reset"
			if strings.Contains(r.URL.Path, "vpngate") {
				kind = "vpngate-reconnect"
			}
		}
		if r.URL.Path == "/api/mihomo/vpngate/install" {
			kind = "vpngate-install"
		}
		task, err := f.store.Enqueue(id, kind, fleet.ID(), payload, 10*time.Minute)
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		writeJSON(w, 202, apiResponse{OK: true, Data: task})
		return
	}
	if r.URL.Path == "/api/state" {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{})
			return
		}
		m := editor.manager
		m.mu.Lock()
		state := copyState(m.state)
		m.mu.Unlock()
		state.Settings = state.Settings.maskSecrets()
		a.fleetSubscriptionView(&state)
		state.LegacyMigrations = nil
		state.WARP = nil
		basics := effectiveMihomoBasics(state)
		state.MihomoBasics = &basics
		state.Traffic = TrafficTotals{}
		n := f.store.Node(id)
		for i := range state.Inbounds {
			in := &state.Inbounds[i]
			if u, ok := n.Status.Traffic[in.ID]; ok {
				in.Traffic = fleetUsageStats(u)
			}
			state.Traffic.Up += in.Traffic.Up
			state.Traffic.Down += in.Traffic.Down
			state.Traffic.AllTimeUp += in.Traffic.AllTimeUp
			state.Traffic.AllTimeDown += in.Traffic.AllTimeDown
			state.Traffic.LegacyCombined += in.Traffic.LegacyCombined
			state.Traffic.AllTimeLegacyCombined += in.Traffic.AllTimeLegacyCombined
			for j := range in.Clients {
				c := &in.Clients[j]
				if u, ok := n.Status.Traffic[in.ID+":"+c.ID]; ok {
					c.Traffic = fleetUsageStats(u)
					c.LastOnline = u.LastOnline
				}
			}
		}
		var details map[string]any
		_ = json.Unmarshal(n.Status.Details, &details)
		if details == nil {
			details = map[string]any{}
		}
		if totals, ok := details["totals"]; ok {
			b, _ := json.Marshal(totals)
			_ = json.Unmarshal(b, &state.Traffic)
		}
		if traffic, ok := details["outboundTraffic"]; ok {
			b, _ := json.Marshal(traffic)
			_ = json.Unmarshal(b, &state.OutboundTraffic)
		}
		details["state"] = state
		details["running"] = n.Status.Running
		details["version"] = n.Status.Version
		details["fleetNode"] = n.ID
		details["lastSeen"] = n.LastSeen
		details["nodeName"] = n.Name
		details["nodeAddress"] = n.EffectiveAddress()
		details["online"] = n.Online(time.Now())
		writeJSON(w, 200, apiResponse{OK: true, Data: details})
		return
	}
	if a.remoteSnapshotResponse(w, r, id) {
		return
	}
	if a.remoteTaskResponse(w, r, id, editor) {
		return
	}
	if r.URL.Path == "/api/subscriptions/token" {
		a.handleFleetNodeSubscriptionToken(w, r, editor)
		return
	}
	handlers := map[string]http.HandlerFunc{
		"/api/config": editor.handleConfig, "/api/backup": editor.handleBackup,
		"/api/subscriptions/token": editor.handleSubscriptionToken,
		"/api/inbounds":            editor.handleInbounds, "/api/inbounds/import": editor.handleInboundImport,
		"/api/clients": editor.handleClients, "/api/raw-config": editor.handleRawConfig,
		"/api/config/download": editor.handleConfigDownload, "/api/traffic/reset": editor.handleTrafficReset,
		"/api/mihomo/settings": editor.handleMihomoSettings, "/api/mihomo/outbound/parse": editor.handleMihomoOutboundYAML,
		"/api/core/logs": editor.handleCoreLogs, "/api/import/xray": editor.handleXrayImport,
		"/api/tools/uuid": editor.handleNewUUID, "/api/tools/inbound-port": editor.handleAvailableInboundPort,
		"/api/tools/subscription-path": editor.handleNewSubscriptionPath,
		"/api/tools/reality-keypair":   editor.handleRealityKeyPair, "/api/tools/vless-encryption": editor.handleVLESSEncryption,
		"/api/tools/ech-keypair": editor.handleECHKeyPair,
	}
	if strings.HasPrefix(r.URL.Path, "/api/mihomo/warp") {
		handlers[r.URL.Path] = editor.handleWarp
	}
	if strings.HasPrefix(r.URL.Path, "/api/mihomo/vpngate") {
		if r.Method == "GET" {
			n := f.store.Node(id)
			var details map[string]json.RawMessage
			_ = json.Unmarshal(n.Status.Details, &details)
			if view := details["vpngate"]; view != nil {
				writeJSON(w, 200, apiResponse{OK: true, Data: view})
				return
			}
		}
		handlers[r.URL.Path] = editor.handleVPNGate
	}
	handler := handlers[r.URL.Path]
	if handler == nil {
		http.NotFound(w, r)
		return
	}
	captured := &captureResponse{header: make(http.Header)}
	handler(captured, r)
	if captured.code < 300 && r.Method != "GET" && r.Method != "HEAD" && !strings.HasPrefix(r.URL.Path, "/api/tools/") {
		if _, err = f.publish(id, editor, nil); err != nil {
			writeJSON(w, 400, apiResponse{Message: "draft saved but publish failed: " + err.Error()})
			return
		}
		if err = a.refreshFleetSubscriptions(); err != nil {
			writeJSON(w, 500, apiResponse{Message: err.Error()})
			return
		}
	}
	for key, values := range captured.header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	if captured.code == 0 {
		captured.code = 200
	}
	w.WriteHeader(captured.code)
	_, _ = w.Write(captured.body.Bytes())
}

func (a *App) handleFleet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.fleet == nil {
		a.handleNodeFleet(w, r)
		return
	}
	f := a.fleet
	path := strings.TrimPrefix(r.URL.Path, "/api/fleet")
	path = strings.Trim(path, "/")
	respond := func(data any, err error) {
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
		} else {
			writeJSON(w, 200, apiResponse{OK: true, Data: data})
		}
	}
	if path == "certificates" || strings.HasPrefix(path, "certificates/") {
		a.handleFleetCertificates(w, r, strings.TrimPrefix(strings.TrimPrefix(path, "certificates"), "/"))
		return
	}
	switch path {
	case "update":
		a.handleFleetUpdate(w, r)
	case "update/check":
		a.handleFleetRelease(w, r)
	case "":
		if r.Method != "GET" {
			writeJSON(w, 405, apiResponse{Message: "GET required"})
			return
		}
		nodes := f.store.Nodes()
		views := []map[string]any{}
		for _, n := range nodes {
			views = append(views, map[string]any{"id": n.ID, "name": n.Name, "address": n.EffectiveAddress(), "configuredAddress": n.Address, "seenIP": n.SeenIP, "addressSource": n.AddressSource(), "revoked": n.Revoked, "online": n.Online(time.Now()), "lastSeen": n.LastSeen, "status": n.Status, "needsUpgrade": n.NeedsUpgrade, "desiredRevision": func() uint64 {
				if n.Desired != nil {
					return n.Desired.Revision
				}
				return 0
			}()})
		}
		respond(map[string]any{"role": "control", "publicKey": f.store.Identity.Public(), "fingerprint": fleet.Fingerprint(f.store.Identity.Public()), "nodes": views, "tasks": f.store.Tasks()}, nil)
	case "tokens":
		if r.Method != "POST" {
			writeJSON(w, 405, apiResponse{})
			return
		}
		var p struct {
			Name   string `json:"name"`
			NodeID string `json:"nodeId"`
		}
		if err := decodeJSON(r, &p); err != nil {
			respond(nil, err)
			return
		}
		token, err := f.store.IssueToken(p.Name, p.NodeID)
		respond(map[string]any{"token": token, "expiresIn": 900, "controlKey": f.store.Identity.Public(), "installerVersion": appVersion}, err)
	case "tokens/revoke":
		if r.Method != "POST" {
			writeJSON(w, 405, apiResponse{})
			return
		}
		var p struct {
			Token string `json:"token"`
		}
		if err := decodeJSON(r, &p); err != nil {
			respond(nil, err)
			return
		}
		respond(nil, f.store.RevokeToken(p.Token))
	case "nodes":
		if r.Method != "POST" {
			writeJSON(w, 405, apiResponse{})
			return
		}
		var p struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		}
		if err := decodeJSON(r, &p); err != nil {
			respond(nil, err)
			return
		}
		n, err := f.store.AddNode(p.Name, p.Address)
		respond(n, err)
	case "tasks":
		if r.Method == "GET" {
			respond(f.store.Tasks(), nil)
			return
		}
		if r.Method != "POST" {
			writeJSON(w, 405, apiResponse{})
			return
		}
		var p struct {
			NodeID  string          `json:"nodeId"`
			Kind    string          `json:"kind"`
			Key     string          `json:"idempotencyKey"`
			Payload json.RawMessage `json:"payload"`
			TTL     int             `json:"ttlSeconds"`
		}
		if err := decodeJSON(r, &p); err != nil {
			respond(nil, err)
			return
		}
		if p.Key == "" {
			p.Key = fleet.ID()
		}
		if p.TTL == 0 {
			p.TTL = 600
		}
		task, err := f.store.Enqueue(p.NodeID, p.Kind, p.Key, p.Payload, time.Duration(p.TTL)*time.Second)
		respond(task, err)
	case "audit":
		if r.Method != "GET" {
			writeJSON(w, 405, apiResponse{})
			return
		}
		respond(f.store.Audits(), nil)
	case "backup":
		a.handleFleetBackup(w, r)
	case "batch":
		a.handleFleetBatch(w, r)
	case "import/merit":
		a.handleMeritImport(w, r)
	case "subscriptions":
		if r.Method != "GET" {
			writeJSON(w, 405, apiResponse{})
			return
		}
		if err := a.refreshFleetSubscriptions(); err != nil {
			respond(nil, err)
			return
		}
		a.manager.mu.Lock()
		tokens := map[string]string{}
		for user, t := range a.manager.state.SubscriptionTokens {
			tokens[user] = t
		}
		a.manager.mu.Unlock()
		respond(tokens, nil)
	default:
		parts := strings.Split(path, "/")
		if len(parts) < 2 || parts[0] != "nodes" {
			http.NotFound(w, r)
			return
		}
		id := parts[1]
		if len(parts) == 2 {
			if r.Method == "GET" {
				n := f.store.Node(id)
				if n == nil {
					http.NotFound(w, r)
					return
				}
				var draft *State
				if !n.Revoked {
					f.mu.Lock()
					ed, err := f.editor(id)
					if err == nil {
						ed.manager.mu.Lock()
						state := copyState(ed.manager.state)
						ed.manager.mu.Unlock()
						state.Settings = state.Settings.maskSecrets()
						draft = &state
					}
					f.mu.Unlock()
					if err != nil {
						respond(nil, err)
						return
					}
				}
				respond(struct {
					*control.Node
					Draft *State `json:"draft,omitempty"`
				}{n, draft}, nil)
				return
			}
			if r.Method == "PUT" {
				var p struct {
					Name    string `json:"name"`
					Address string `json:"address"`
				}
				if err := decodeJSON(r, &p); err != nil {
					respond(nil, err)
					return
				}
				respond(nil, f.store.UpdateNode(id, p.Name, p.Address))
				return
			}
		}
		if len(parts) != 3 || r.Method != "POST" {
			writeJSON(w, 405, apiResponse{})
			return
		}
		switch parts[2] {
		case "sidecar":
			var input fleetSidecarInput
			if err := decodeJSON(r, &input); err != nil {
				respond(nil, err)
				return
			}
			f.mu.Lock()
			state, err := f.createSidecar(id, input)
			f.mu.Unlock()
			respond(state, err)
		case "revoke":
			respond(nil, f.store.Revoke(id))
		case "publish":
			f.mu.Lock()
			editor, err := f.editor(id)
			var state *fleet.DesiredState
			if err == nil {
				state, err = f.publish(id, editor, nil)
			}
			f.mu.Unlock()
			if err == nil {
				err = a.refreshFleetSubscriptions()
			}
			respond(state, err)
		case "model":
			var state State
			if err := decodeJSON(r, &state); err != nil {
				respond(nil, err)
				return
			}
			for _, in := range state.Inbounds {
				if err := validateInbound(in); err != nil {
					respond(nil, err)
					return
				}
			}
			f.mu.Lock()
			editor, err := f.editor(id)
			var desired *fleet.DesiredState
			if err == nil {
				editor.manager.mu.Lock()
				editor.manager.state = state
				err = editor.manager.saveLocked()
				editor.manager.mu.Unlock()
				if err == nil {
					desired, err = f.publish(id, editor, nil)
				}
			}
			f.mu.Unlock()
			if err == nil {
				err = a.refreshFleetSubscriptions()
			}
			respond(desired, err)
		case "sidecars":
			var sidecars []fleet.Sidecar
			if err := decodeJSON(r, &sidecars); err != nil {
				respond(nil, err)
				return
			}
			for _, s := range sidecars {
				if !fleet.SafeName(s.ID) || !json.Valid([]byte(s.Config)) {
					respond(nil, errors.New("invalid sidecar"))
					return
				}
			}
			f.mu.Lock()
			editor, err := f.editor(id)
			var state *fleet.DesiredState
			if err == nil {
				state, err = f.publish(id, editor, sidecars)
			}
			f.mu.Unlock()
			respond(state, err)
		default:
			http.NotFound(w, r)
		}
	}
}

func (a *App) handleNodeFleet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/fleet/import/merit" {
		a.handleStandaloneImport(w, r)
		return
	}
	if r.URL.Path == "/api/fleet/sidecars" {
		a.handleStandaloneSidecars(w, r)
		return
	}
	if strings.TrimRight(r.URL.Path, "/") == "/api/fleet" && r.Method == "GET" {
		data := map[string]any{"role": "node", "managed": a.nodeRuntime != nil}
		if a.nodeRuntime != nil {
			data["status"] = a.nodeRuntime.Snapshot()
		}
		writeJSON(w, 200, apiResponse{OK: true, Data: data})
		return
	}
	if r.URL.Path == "/api/fleet/break-glass" && r.Method == "POST" {
		err := fleet.WriteFile(filepath.Join(a.manager.dataDir, "break-glass"), []byte(time.Now().UTC().Format(time.RFC3339)), 0600)
		if err != nil {
			writeJSON(w, 500, apiResponse{Message: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{OK: true})
		return
	}
	if r.URL.Path == "/api/fleet/join" && r.Method == "POST" {
		var p struct {
			URL   string `json:"url"`
			Key   string `json:"key"`
			Token string `json:"token"`
			Name  string `json:"name"`
		}
		if err := decodeJSON(r, &p); err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		_, err := agent.Enroll(r.Context(), a.manager.dataDir, p.URL, p.Key, p.Token, p.Name)
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		writeJSON(w, 200, apiResponse{OK: true, Message: "enrolled; restart node to activate management"})
		return
	}
	http.NotFound(w, r)
}
