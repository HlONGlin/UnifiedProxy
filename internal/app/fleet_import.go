package app

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hlonglin/unifiedproxy/internal/agent"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type meritPort struct {
	ID              string `json:"id"`
	Port            int    `json:"port"`
	Protocol        string `json:"protocol"`
	ProxyType       string `json:"proxyType"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Label           string `json:"label"`
	Enabled         bool   `json:"enabled"`
	TrafficLimit    int64  `json:"trafficLimit"`
	TrafficUsed     int64  `json:"trafficUsed"`
	TrafficResetAt  string `json:"trafficResetAt"`
	CreatedAt       string `json:"createdAt"`
	AutoDisabled    bool   `json:"autoDisabled"`
	BandwidthLimit  int64  `json:"bandwidthLimit"`
	TLSFingerprint  string `json:"tlsFingerprint"`
	TLSCert         string `json:"tlsCert"`
	TLSKey          string `json:"tlsKey"`
	Domain          string `json:"domain"`
	AutoACME        bool   `json:"autoACME"`
	HandshakeAddr   string `json:"handshakeAddr"`
	BackendAddr     string `json:"backendAddr"`
	BackendPassword string `json:"backendPassword"`
	BackendCipher   string `json:"backendCipher"`
	BackendManaged  bool   `json:"backendManaged"`
}
type meritNode struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Address   string       `json:"address"`
	Domain    string       `json:"domain"`
	Ports     []meritPort  `json:"ports"`
	Remark    string       `json:"remark"`
	CreatedAt string       `json:"createdAt"`
	Egress    *meritEgress `json:"egress"`
}
type meritImportResult struct {
	Nodes []map[string]string `json:"nodes"`
	Log   []string            `json:"log"`
}

func convertMerit(node meritNode) (State, []fleet.Sidecar, []string, error) {
	state := defaultState()
	sidecars := []fleet.Sidecar{}
	logs := []string{}
	if node.ID == "" {
		return state, nil, nil, errors.New("legacy node ID is required")
	}
	seenIDs, seenPorts := map[string]bool{}, map[string]bool{}
	for _, p := range node.Ports {
		// The unsupported legacy Naive form is recovery-only. Validate supported
		// siblings independently, even when that record has unusable credentials.
		if p.ProxyType == "naive" && (p.Port < 1 || p.Port > 65535 || p.Username == "" || p.Password == "" || p.Domain == "" || (!p.AutoACME && (!strings.Contains(p.TLSCert, "BEGIN CERTIFICATE") || !strings.Contains(p.TLSKey, "PRIVATE KEY")))) {
			logs = append(logs, fmt.Sprintf("NaiveProxy %d excluded from activation; original record retained in recovery archive", p.Port))
			continue
		}
		if p.ID != "" && seenIDs[p.ID] {
			return state, nil, nil, errors.New("duplicate legacy port ID")
		}
		seenIDs[p.ID] = true
		transport := strings.ToUpper(valueOr(p.Protocol, "TCP"))
		if p.ProxyType != "" && p.ProxyType != "mieru" {
			transport = "TCP"
		}
		portKey := transport + ":" + strconv.Itoa(p.Port)
		if seenPorts[portKey] {
			return state, nil, nil, errors.New("duplicate legacy listen port")
		}
		seenPorts[portKey] = true
		if p.Port < 1 || p.Port > 65535 || p.Password == "" {
			return state, nil, nil, fmt.Errorf("invalid port %d", p.Port)
		}
		if p.TrafficUsed < 0 || p.TrafficLimit < 0 || p.BandwidthLimit < 0 {
			return state, nil, nil, errors.New("negative legacy traffic or limit")
		}
		typeName := p.ProxyType
		if typeName == "" {
			typeName = "mieru"
		}
		h := sha256.Sum256([]byte(node.ID + ":" + p.ID + ":" + strconv.Itoa(p.Port)))
		id := hex.EncodeToString(h[:8])
		if typeName == "mieru" || typeName == "anytls" {
			client := Client{ID: "client-" + id, Name: valueOr(p.Username, "user-"+id), Username: p.Username, Password: p.Password, Enabled: true}
			if p.TrafficLimit > 0 {
				client.Total = uint64(p.TrafficLimit)
			}
			client.CreatedAt = p.CreatedAt
			client.Traffic = TrafficStats{LegacyCombined: uint64(p.TrafficUsed), AllTimeLegacyCombined: uint64(p.TrafficUsed)}
			in := Inbound{ID: id, Name: xrayImportName(valueOr(p.Label, typeName+"-"+strconv.Itoa(p.Port))) + "-" + id[:6], Type: typeName, Listen: "0.0.0.0", Port: p.Port, Enabled: p.Enabled || p.AutoDisabled, Clients: []Client{client}, TrafficReset: "monthly", CreatedAt: p.CreatedAt, Notes: node.Remark, LastTrafficReset: p.TrafficResetAt, Traffic: client.Traffic, LegacySeed: &legacyTrafficSeed{Combined: uint64(p.TrafficUsed), ResetAt: p.TrafficResetAt}}
			if p.TrafficUsed > 0 {
				logs = append(logs, fmt.Sprintf("%s: %d combined legacy bytes retained separately; directional history was never recorded", id, p.TrafficUsed))
			}
			if p.BandwidthLimit > 0 {
				logs = append(logs, fmt.Sprintf("%s: bandwidthLimit=%d retained in recovery archive; legacy versions stored but did not enforce this field", id, p.BandwidthLimit))
			}
			if typeName == "mieru" {
				in.Mieru.Transport = strings.ToUpper(valueOr(p.Protocol, "TCP"))
			} else {
				in.TLS = true
				in.TLSServerName = p.Domain
				in.TLSFingerprint = p.TLSFingerprint
				in.Certificate = p.TLSCert
				in.PrivateKey = p.TLSKey
				if !strings.Contains(in.Certificate, "BEGIN CERTIFICATE") || !strings.Contains(in.PrivateKey, "PRIVATE KEY") {
					logs = append(logs, fmt.Sprintf("AnyTLS %d needs a centrally managed certificate before publishing", p.Port))
					in.Enabled = false
					in.Certificate = ""
					in.PrivateKey = ""
				}
			}
			state.Inbounds = append(state.Inbounds, in)
			logs = append(logs, fmt.Sprintf("%s %d -> listener %s; original credentials preserved", typeName, p.Port, id))
			continue
		}
		if typeName == "naive" {
			if p.Domain == "" || (!p.AutoACME && (!strings.Contains(p.TLSCert, "BEGIN CERTIFICATE") || !strings.Contains(p.TLSKey, "PRIVATE KEY"))) {
				logs = append(logs, fmt.Sprintf("NaiveProxy %d excluded from activation; original record retained in recovery archive", p.Port))
				continue
			}
			credentials := base64.StdEncoding.EncodeToString([]byte(p.Username + ":" + p.Password))
			tlsApp := map[string]any{}
			if p.AutoACME {
				tlsApp["certificates"] = map[string]any{"automate": []string{p.Domain}}
				logs = append(logs, "NaiveProxy retains its existing Caddy ACME policy; migrate it to a central certificate when available")
			} else {
				tlsApp["certificates"] = map[string]any{"load_pem": []any{map[string]string{"certificate": p.TLSCert, "key": p.TLSKey}}}
				if !strings.Contains(p.TLSCert, "BEGIN CERTIFICATE") {
					return state, nil, nil, errors.New("NaiveProxy migration requires PEM certificate content, not a path on the old node")
				}
			}
			cfg := map[string]any{"admin": map[string]bool{"disabled": true}, "apps": map[string]any{"tls": tlsApp, "http": map[string]any{"servers": map[string]any{"naive": map[string]any{"listen": []string{":" + strconv.Itoa(p.Port)}, "routes": []any{map[string]any{"handle": []any{map[string]any{"handler": "forward_proxy", "hide_ip": true, "hide_via": true, "auth_credentials": [][]byte{[]byte(credentials)}, "probe_resistance": map[string]any{}}}}}, "tls_connection_policies": []any{map[string]any{}}, "automatic_https": map[string]bool{"disable_redirects": true}}}}}}
			b, _ := json.Marshal(cfg)
			sidecars = append(sidecars, fleet.Sidecar{ID: id, Kind: "naive", Enabled: p.Enabled, Config: string(b), Address: p.Domain, Port: p.Port, Username: p.Username, Password: p.Password, ServerName: p.Domain, Metering: "unmetered"})
			continue
		}
		if typeName == "shadowtls" {
			if p.BackendAddr == "" || p.BackendPassword == "" {
				return state, nil, nil, errors.New("ShadowTLS backend address and password are required")
			}
			cfg := agent.ShadowConfig{Listen: net.JoinHostPort("0.0.0.0", strconv.Itoa(p.Port)), Backend: p.BackendAddr, Handshake: valueOr(p.HandshakeAddr, "cloudflare.com:443"), Password: p.Password}
			if p.BackendManaged {
				host, port, err := net.SplitHostPort(p.BackendAddr)
				ip := net.ParseIP(host)
				if err != nil || ip == nil || !ip.IsLoopback() {
					return state, nil, nil, errors.New("managed ShadowTLS backend must use loopback")
				}
				num, err := strconv.Atoi(port)
				if err != nil {
					return state, nil, nil, err
				}
				b, _ := json.Marshal(map[string]any{"server": host, "server_port": num, "password": p.BackendPassword, "method": valueOr(p.BackendCipher, "aes-256-gcm"), "mode": "tcp_only"})
				sidecars = append(sidecars, fleet.Sidecar{ID: id + "-backend", Kind: "shadowsocks", Enabled: p.Enabled, Config: string(b), Metering: "unmetered"})
			}
			b, _ := json.Marshal(cfg)
			serverName, _, _ := net.SplitHostPort(cfg.Handshake)
			sidecars = append(sidecars, fleet.Sidecar{ID: id, Kind: "shadowtls", Enabled: p.Enabled, Config: string(b), Port: p.Port, Username: valueOr(p.Username, "shadowtls-"+id), Password: p.Password, ServerName: valueOr(p.Domain, serverName), BackendPassword: p.BackendPassword, Cipher: valueOr(p.BackendCipher, "aes-256-gcm"), Metering: "unmetered"})
			continue
		}
		return state, nil, nil, fmt.Errorf("unsupported legacy proxy type %q", typeName)
	}
	if err := convertMeritEgress(node, &state); err != nil {
		return state, nil, nil, err
	}
	return state, sidecars, logs, nil
}

func (a *App) handleMeritImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	input, raw, err := decodeMeritBackup(r.Body)
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	type converted struct {
		node     meritNode
		state    State
		sidecars []fleet.Sidecar
		log      []string
	}
	plans := []converted{}
	seen := map[string]bool{}
	for _, node := range input.Nodes {
		if seen[node.ID] {
			writeJSON(w, 400, apiResponse{Message: "duplicate legacy node ID"})
			return
		}
		seen[node.ID] = true
		state, sidecars, logs, err := convertMerit(node)
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		if _, err = renderStateConfigAt(state, 0); err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		plans = append(plans, converted{node, state, sidecars, logs})
	}
	f := a.fleet
	f.mu.Lock()
	defer f.mu.Unlock()
	before, err := a.exportFleetBackup()
	if err != nil {
		writeJSON(w, 500, apiResponse{Message: err.Error()})
		return
	}
	journal := filepath.Join(f.dataDir, "fleet-restore-transaction.json")
	if err = fleet.WriteFile(journal, before, 0600); err != nil {
		writeJSON(w, 500, apiResponse{Message: err.Error()})
		return
	}
	rollback := func(cause error) {
		if restoreErr := a.restoreFleetBackupTransaction(before, false); restoreErr != nil {
			cause = fmt.Errorf("import: %v; rollback: %w", cause, restoreErr)
		} else {
			_ = os.Remove(journal)
		}
		writeJSON(w, 500, apiResponse{Message: cause.Error()})
	}
	result := meritImportResult{Log: []string{"Current administrator and panel path retained. Source-only fields are preserved in the protected backup recovery archive.", "Old API keys, transient tasks and cloud credentials are not activated; enroll each node with a fresh node-specific token."}}
	ids := []string{}
	for _, plan := range plans {
		hash := sha256.Sum256([]byte("merit:" + plan.node.ID))
		n, created, err := f.store.ImportNode("merit-"+hex.EncodeToString(hash[:16]), plan.node.Name, valueOr(plan.node.Address, plan.node.Domain))
		if err != nil {
			rollback(err)
			return
		}
		ids = append(ids, n.ID)
		if created {
			editor, err := f.editor(n.ID)
			if err == nil {
				editor.manager.mu.Lock()
				editor.manager.state = plan.state
				err = editor.manager.saveLocked()
				editor.manager.mu.Unlock()
			}
			if err == nil {
				_, err = f.publish(n.ID, editor, plan.sidecars)
			}
			if err != nil {
				rollback(err)
				return
			}
		} else {
			result.Log = append(result.Log, "Already imported node "+plan.node.ID+"; existing edits preserved.")
		}
		result.Nodes = append(result.Nodes, map[string]string{"oldNodeId": plan.node.ID, "nodeId": n.ID, "oldSubscription": "retained when settings.subToken is present", "newSubscription": "GET /api/fleet/subscriptions"})
		result.Log = append(result.Log, plan.log...)
	}
	a.manager.mu.Lock()
	appendLegacyMigration(&a.manager.state, raw, ids)
	err = a.manager.saveLocked()
	a.manager.mu.Unlock()
	if err == nil {
		err = a.refreshFleetSubscriptions()
	}
	if err != nil {
		rollback(err)
		return
	}
	if err = os.Remove(journal); err != nil {
		rollback(err)
		return
	}
	writeJSON(w, 200, apiResponse{OK: true, Data: result})
}
