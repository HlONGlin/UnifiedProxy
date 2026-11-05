package app

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

// Recovery-only source data is included in protected backups, never in the
// ordinary state API or a node's desired model. Old tasks/keys are not activated.
type legacyMigration struct {
	ID         string          `json:"id"`
	Source     json.RawMessage `json:"source"`
	NodeIDs    []string        `json:"nodeIds"`
	ImportedAt string          `json:"importedAt"`
}
type legacyTrafficSeed struct {
	Combined uint64 `json:"combined"`
	ResetAt  string `json:"resetAt,omitempty"`
}
type meritBackup struct {
	Nodes    []meritNode `json:"nodes"`
	Settings struct {
		SubToken  string `json:"subToken"`
		PanelPath string `json:"panelPath"`
	} `json:"settings"`
}

func decodeMeritBackup(r io.Reader) (meritBackup, []byte, error) {
	var b meritBackup
	raw, err := io.ReadAll(io.LimitReader(r, (64<<20)+1))
	if err != nil {
		return b, nil, err
	}
	if len(raw) > 64<<20 {
		return b, nil, errors.New("legacy backup exceeds 64 MiB")
	}
	if err = json.Unmarshal(raw, &b); err != nil {
		return b, nil, err
	}
	if b.Nodes == nil || len(b.Nodes) > 500 {
		return b, nil, errors.New("backup requires a nodes array with at most 500 nodes")
	}
	if p := b.Settings.PanelPath; p != "" && (strings.ContainsAny(p, "?#\\") || path.Clean("/"+strings.Trim(p, "/")) != "/"+strings.Trim(p, "/")) {
		return b, nil, errors.New("invalid legacy panel path")
	}
	return b, raw, nil
}

func appendLegacyMigration(s *State, raw []byte, nodes []string) {
	var compact any
	_ = json.Unmarshal(raw, &compact)
	normalized, _ := json.Marshal(compact)
	sum := sha256.Sum256(normalized)
	id := hex.EncodeToString(sum[:])
	for i := range s.LegacyMigrations {
		old := &s.LegacyMigrations[i]
		if old.ID == id {
			seen := make(map[string]bool, len(old.NodeIDs))
			for _, node := range old.NodeIDs {
				seen[node] = true
			}
			for _, node := range nodes {
				if !seen[node] {
					old.NodeIDs = append(old.NodeIDs, node)
					seen[node] = true
				}
			}
			return
		}
	}
	s.LegacyMigrations = append(s.LegacyMigrations, legacyMigration{ID: id, Source: append(json.RawMessage(nil), raw...), NodeIDs: nodes, ImportedAt: time.Now().UTC().Format(time.RFC3339)})
}

func (t TrafficStats) used() uint64 { return t.Up + t.Down + t.LegacyCombined }
func subscriptionLegacyBytes(clients []subscriptionClient) uint64 {
	var total uint64
	for _, item := range clients {
		total += item.Client.Traffic.LegacyCombined
	}
	return total
}
func (t TrafficStats) allTime() uint64 { return t.AllTimeUp + t.AllTimeDown + t.AllTimeLegacyCombined }

func (a *App) importJSONBackup(w http.ResponseWriter, r *http.Request, source io.Reader) {
	raw, err := io.ReadAll(io.LimitReader(source, (64<<20)+1))
	if err != nil || len(raw) > 64<<20 {
		writeJSON(w, 400, apiResponse{Message: "invalid/oversized JSON backup"})
		return
	}
	var shape map[string]json.RawMessage
	if err = json.Unmarshal(raw, &shape); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	if shape["nodes"] != nil {
		copyRequest := r.Clone(r.Context())
		copyRequest.Body = io.NopCloser(bytes.NewReader(raw))
		if a.fleet != nil {
			a.handleMeritImport(w, copyRequest)
		} else {
			a.handleStandaloneImport(w, copyRequest)
		}
		return
	}
	if shape["fleet"] != nil && a.fleet != nil {
		var backup fleetBackup
		if err = json.Unmarshal(raw, &backup); err == nil {
			if r.FormValue("replaceIdentity") != "1" {
				a.manager.mu.Lock()
				current := a.manager.state.Settings
				a.manager.mu.Unlock()
				backup.Panel.Settings.Username = current.Username
				backup.Panel.Settings.Password = current.Password
				backup.Panel.Settings.PanelPath = current.PanelPath
			}
			raw, err = json.Marshal(backup)
		}
		if err == nil {
			a.fleet.mu.Lock()
			err = a.restoreFleetBackup(raw)
			a.fleet.mu.Unlock()
		}
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
		} else {
			writeJSON(w, 200, apiResponse{OK: true, Message: "Fleet backup restored; restart control to reload auxiliary registries"})
		}
		return
	}
	writeJSON(w, 400, apiResponse{Message: "unrecognized JSON backup; use a legacy nodes store, fleet backup or UnifiedProxy ZIP v1/v2"})
}

// Only a new listener receives the historical seed. Existing listeners keep
// their live counters, including an intentional reset to zero.
func seedLegacyTraffic(incoming *State, old State) {
	seen := map[string]bool{}
	for _, in := range old.Inbounds {
		seen[in.ID] = true
	}
	for i := range incoming.Inbounds {
		in := &incoming.Inbounds[i]
		if in.LegacySeed == nil || seen[in.ID] {
			continue
		}
		seed := in.LegacySeed
		in.Traffic.LegacyCombined = seed.Combined
		in.Traffic.AllTimeLegacyCombined = seed.Combined
		in.LastTrafficReset = seed.ResetAt
		for j := range in.Clients {
			c := &in.Clients[j]
			c.Traffic.LegacyCombined = seed.Combined
			c.Traffic.AllTimeLegacyCombined = seed.Combined
			c.LastTrafficReset = seed.ResetAt
		}
	}
}

func (a *App) handleLegacySubscription(w http.ResponseWriter, r *http.Request) bool {
	a.manager.mu.Lock()
	archives := append([]legacyMigration(nil), a.manager.state.LegacyMigrations...)
	settings := a.manager.state.Settings
	a.manager.mu.Unlock()
	matched := false
	for _, archive := range archives {
		var b meritBackup
		if json.Unmarshal(archive.Source, &b) != nil || b.Settings.SubToken == "" {
			continue
		}
		oldPath := strings.TrimRight(b.Settings.PanelPath, "/") + "/sub"
		if !strings.HasPrefix(oldPath, "/") {
			oldPath = "/" + oldPath
		}
		if r.URL.Path != oldPath {
			continue
		}
		matched = true
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return true
		}
		if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("token")), []byte(b.Settings.SubToken)) != 1 {
			continue
		}
		snapshot := subscriptionSnapshot{Settings: settings}
		allowed := map[string]bool{}
		for _, id := range archive.NodeIDs {
			allowed[id] = true
		}
		if a.fleet != nil {
			for _, node := range a.fleet.store.Nodes() {
				if !allowed[node.ID] || node.Revoked || node.Desired == nil || node.Address == "" {
					continue
				}
				var state State
				if json.Unmarshal(node.Desired.Model, &state) != nil {
					continue
				}
				seedLegacyTraffic(&state, State{})
				for i := range state.Inbounds {
					in := &state.Inbounds[i]
					for j := range in.Clients {
						c := &in.Clients[j]
						if usage, ok := node.Status.Traffic[in.ID+":"+c.ID]; ok {
							c.Traffic = TrafficStats{Up: usage.Up, Down: usage.Down, LegacyCombined: usage.LegacyCombined, AllTimeLegacyCombined: usage.AllTimeLegacyCombined}
						}
					}
				}
				for _, in := range state.Inbounds {
					in.Name = node.Name + " / " + in.Name
					for _, c := range in.Clients {
						snapshot.Clients = append(snapshot.Clients, subscriptionClient{Inbound: in, Client: c, Server: node.Address})
					}
				}
				for _, s := range node.Desired.Sidecars {
					if s.Enabled && s.Kind != "naive" {
						snapshot.FleetSidecars = append(snapshot.FleetSidecars, fleetSubscriptionSidecar{NodeName: node.Name, Server: node.Address, Config: s})
					}
				}
			}
		} else {
			a.manager.mu.Lock()
			inbounds := copyState(a.manager.state).Inbounds
			a.manager.mu.Unlock()
			ids := map[string]bool{}
			sidecarIDs := map[string]bool{}
			for _, n := range b.Nodes {
				if allowed[n.ID] {
					converted, configs, _, err := convertMerit(n)
					if err == nil {
						for _, s := range configs {
							sidecarIDs[s.ID] = true
						}
						for _, in := range converted.Inbounds {
							ids[in.ID] = true
						}
					}
				}
			}
			for _, in := range inbounds {
				if ids[in.ID] {
					for _, c := range in.Clients {
						snapshot.Clients = append(snapshot.Clients, subscriptionClient{Inbound: in, Client: c})
					}
				}
			}
			for _, s := range a.localSidecarConfigs() {
				if sidecarIDs[s.ID] && s.Enabled && s.Kind != "naive" {
					snapshot.FleetSidecars = append(snapshot.FleetSidecars, fleetSubscriptionSidecar{NodeName: "local", Config: s})
				}
			}
		}
		data, err := buildMihomoSubscription(snapshot, subscriptionServerHost(r))
		if err != nil {
			http.Error(w, "subscription unavailable", 500)
			return true
		}
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-UnifiedProxy-Legacy", "combined traffic retained without directional history")
		if r.Method != "HEAD" {
			_, _ = w.Write(data)
		}
		return true
	}
	if matched {
		http.Error(w, "invalid token", 403)
		return true
	}
	return false
}
