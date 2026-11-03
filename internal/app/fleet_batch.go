package app

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
)

func (a *App) handleFleetBatch(w http.ResponseWriter, r *http.Request) {
	f := a.fleet
	if r.Method == "GET" {
		writeJSON(w, 200, apiResponse{OK: true, Data: f.store.Rollouts()})
		return
	}
	if r.Method != "POST" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	var p struct {
		Nodes      []string        `json:"nodes"`
		Action     string          `json:"action"`
		Kind       string          `json:"kind"`
		Payload    json.RawMessage `json:"payload"`
		Inbound    Inbound         `json:"inbound"`
		Username   string          `json:"username"`
		Total      uint64          `json:"total"`
		ExpiryTime int64           `json:"expiryTime"`
	}
	if err := decodeJSON(r, &p); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	if len(p.Nodes) == 0 || len(p.Nodes) > 500 {
		writeJSON(w, 400, apiResponse{Message: "select 1-500 nodes"})
		return
	}
	if p.Action == "task" {
		rollout, err := f.store.Rollout(p.Nodes, p.Kind, p.Payload)
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		writeJSON(w, 202, apiResponse{OK: true, Data: rollout})
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	type planned struct {
		id     string
		editor *App
		state  State
	}
	plans := []planned{}
	seen := map[string]bool{}
	for _, id := range p.Nodes {
		if seen[id] {
			writeJSON(w, 400, apiResponse{Message: "duplicate node"})
			return
		}
		seen[id] = true
		editor, err := f.editor(id)
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		editor.manager.mu.Lock()
		state := copyState(editor.manager.state)
		editor.manager.mu.Unlock()
		switch p.Action {
		case "inbound":
			in := p.Inbound
			in.ID = fleet.ID()[:12]
			for j := range in.Clients {
				in.Clients[j].ID = fleet.ID()[:12]
			}
			if err = validateInbound(in); err != nil {
				writeJSON(w, 400, apiResponse{Message: err.Error()})
				return
			}
			for _, old := range state.Inbounds {
				if old.Name == in.Name || old.Port == in.Port {
					writeJSON(w, 400, apiResponse{Message: "inbound name or port conflicts on " + id})
					return
				}
			}
			state.Inbounds = append(state.Inbounds, in)
		case "quota":
			if p.Username == "" {
				writeJSON(w, 400, apiResponse{Message: "username is required"})
				return
			}
			for i := range state.Inbounds {
				for j := range state.Inbounds[i].Clients {
					client := &state.Inbounds[i].Clients[j]
					if subscriptionUsername(*client) == p.Username {
						client.Total = p.Total
						client.ExpiryTime = p.ExpiryTime
					}
				}
			}
		case "defaults":
			a.manager.mu.Lock()
			defaults := copyState(a.manager.state)
			a.manager.mu.Unlock()
			state.Outbounds = defaults.Outbounds
			state.RoutingRules = defaults.RoutingRules
			state.MihomoBasics = defaults.MihomoBasics
			state.WARP = defaults.WARP
			state.Settings.AllowLAN = defaults.Settings.AllowLAN
			state.Settings.Mode = defaults.Settings.Mode
			state.Settings.LogLevel = defaults.Settings.LogLevel
		default:
			writeJSON(w, 400, apiResponse{Message: errors.New("unknown batch action").Error()})
			return
		}
		if _, err = renderStateConfig(state); err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		plans = append(plans, planned{id, editor, state})
	}
	results := []map[string]any{}
	for _, plan := range plans {
		plan.editor.manager.mu.Lock()
		plan.editor.manager.state = plan.state
		err := plan.editor.manager.saveLocked()
		plan.editor.manager.mu.Unlock()
		var desired *fleet.DesiredState
		if err == nil {
			desired, err = f.publish(plan.id, plan.editor, nil)
		}
		if err != nil {
			results = append(results, map[string]any{"nodeId": plan.id, "error": err.Error()})
		} else {
			results = append(results, map[string]any{"nodeId": plan.id, "revision": desired.Revision})
		}
	}
	if err := a.refreshFleetSubscriptions(); err != nil {
		writeJSON(w, 500, apiResponse{Message: err.Error(), Data: results})
		return
	}
	writeJSON(w, 200, apiResponse{OK: true, Data: results})
}
