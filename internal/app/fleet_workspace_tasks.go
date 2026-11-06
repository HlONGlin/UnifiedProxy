package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func (f *fleetConsole) policyTask(id string, desired *fleet.DesiredState) *fleet.Task {
	node := f.store.Node(id)
	if node.Status.Revision == desired.Revision && node.Status.Digest == desired.Digest {
		return nil
	}
	tasks := f.store.Tasks()
	for i := len(tasks) - 1; i >= 0; i-- {
		task := tasks[i]
		if task.NodeID != id || task.Kind != "apply-state" {
			continue
		}
		var state fleet.DesiredState
		if json.Unmarshal(task.Payload, &state) == nil && state.Revision == desired.Revision && state.Digest == desired.Digest {
			return task
		}
	}
	return nil
}

func (a *App) remoteTaskResponse(w http.ResponseWriter, r *http.Request, id string, editor *App) bool {
	if strings.HasPrefix(r.URL.Path, "/api/node-tasks/") {
		if r.Method != http.MethodGet {
			writeJSON(w, 405, apiResponse{})
			return true
		}
		taskID := strings.TrimPrefix(r.URL.Path, "/api/node-tasks/")
		for _, task := range a.fleet.store.Tasks() {
			if task.ID == taskID && task.NodeID == id {
				writeJSON(w, 200, apiResponse{OK: true, Data: task})
				return true
			}
		}
		writeJSON(w, 404, apiResponse{Message: "unknown server task"})
		return true
	}
	if r.URL.Path != "/api/mihomo/outbound/delay" {
		return false
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, apiResponse{})
		return true
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, 400, apiResponse{Message: "invalid outbound"})
		return true
	}
	editor.manager.mu.Lock()
	valid := input.Name == "DIRECT"
	for _, outbound := range editor.manager.state.Outbounds {
		if outbound.Name == input.Name && !containsString([]string{"reject", "reject-drop"}, strings.ToLower(strings.TrimSpace(outbound.Type))) {
			valid = true
		}
	}
	editor.manager.mu.Unlock()
	if !valid {
		writeJSON(w, 400, apiResponse{Message: a.tr("Outbound 不存在或不支持测试")})
		return true
	}
	n := a.fleet.store.Node(id)
	if !n.Online(time.Now()) || !n.Status.Running {
		writeJSON(w, 409, apiResponse{Message: "子服务器离线或 Mihomo 尚未启动"})
		return true
	}
	if !fleet.Supports(n.Status.Capabilities, "outbound-delay") {
		writeJSON(w, 409, apiResponse{Message: "请先升级此子服务器，以支持远程出口延迟测试"})
		return true
	}
	body, _ := json.Marshal(input)
	task, err := a.fleet.store.Enqueue(id, "outbound-delay", fleet.ID(), body, time.Minute)
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return true
	}
	writeJSON(w, 202, apiResponse{OK: true, Data: task})
	return true
}
