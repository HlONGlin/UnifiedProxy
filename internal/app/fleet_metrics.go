package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func (a *App) fleetSubscriptionView(state *State) {
	a.manager.mu.Lock()
	defer a.manager.mu.Unlock()
	host := a.manager.state.Settings
	state.Settings.PanelDomain = host.PanelDomain
	state.Settings.SubscriptionPort = host.SubscriptionPort
	state.Settings.SubscriptionPath = host.SubscriptionPath
	state.Settings.ClashPath = host.ClashPath
	state.Settings.SubscriptionClients = host.SubscriptionClients
	state.SubscriptionTokens = map[string]string{}
	for _, client := range allStateClients(state.Inbounds) {
		user := subscriptionUsername(client)
		if token := a.manager.state.SubscriptionTokens[user]; token != "" {
			state.SubscriptionTokens[user] = token
		}
	}
}

func (a *App) handleFleetNodeSubscriptionToken(w http.ResponseWriter, r *http.Request, editor *App) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, apiResponse{})
		return
	}
	var input struct {
		Username string `json:"username"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, 400, apiResponse{Message: "invalid request"})
		return
	}
	editor.manager.mu.Lock()
	found := false
	for _, client := range allStateClients(editor.manager.state.Inbounds) {
		if subscriptionUsername(client) == strings.TrimSpace(input.Username) {
			found = true
			break
		}
	}
	editor.manager.mu.Unlock()
	if !found {
		writeJSON(w, 404, apiResponse{Message: "user does not belong to this server"})
		return
	}
	body, _ := json.Marshal(input)
	r.Body = io.NopCloser(bytes.NewReader(body))
	a.handleSubscriptionToken(w, r)
}

func fleetUsageStats(u fleet.Usage) TrafficStats {
	return TrafficStats{LegacyCombined: u.LegacyCombined, AllTimeLegacyCombined: u.AllTimeLegacyCombined, Up: u.Up, Down: u.Down, AllTimeUp: u.AllTimeUp, AllTimeDown: u.AllTimeDown}
}

// Older nodes return empty metrics until upgraded; never substitute control-host
// CPU history or logs for missing child telemetry.
func (a *App) remoteSnapshotResponse(w http.ResponseWriter, r *http.Request, id string) bool {
	path := r.URL.Path
	if path != "/api/core/logs" && path != "/api/logs/download" && !strings.HasPrefix(path, "/api/server/cpuHistory/") {
		return false
	}
	if r.Method != http.MethodGet {
		writeJSON(w, 405, apiResponse{})
		return true
	}
	var details struct {
		Logs    []string               `json:"logs"`
		History map[string][]cpuSample `json:"cpuHistory"`
	}
	_ = json.Unmarshal(a.fleet.store.Node(id).Status.Details, &details)
	if strings.HasPrefix(path, "/api/server/cpuHistory/") {
		bucket := strings.TrimPrefix(path, "/api/server/cpuHistory/")
		if !map[string]bool{"2": true, "30": true, "60": true, "120": true, "180": true, "300": true}[bucket] {
			writeJSON(w, 400, apiResponse{Message: "invalid bucket"})
			return true
		}
		points := details.History[bucket]
		if points == nil {
			points = []cpuSample{}
		}
		writeJSON(w, 200, apiResponse{OK: true, Data: points})
		return true
	}
	logs := translateLogs(details.Logs, a.lang())
	if logs == nil {
		logs = []string{}
	}
	if path == "/api/logs/download" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="node-`+id+`.log"`)
		_, _ = io.WriteString(w, strings.Join(logs, "\n")+"\n")
	} else {
		writeJSON(w, 200, apiResponse{OK: true, Data: logs})
	}
	return true
}
