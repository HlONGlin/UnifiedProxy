package app

import (
	"context"
	"encoding/json"
	"github.com/RomanovCaesar/m-ui/internal/agent"
	"github.com/RomanovCaesar/m-ui/internal/fleet"
	"io"
	"net/http"
	"sync"
	"time"
)

type standaloneRuntime struct {
	mu      sync.Mutex
	runtime *nodeRuntime
}

func (a *App) initializeStandalone(ctx context.Context, dir string) error {
	n := &nodeRuntime{manager: a.manager, sidecars: &agent.Sidecars{Dir: dir}}
	if err := n.recover(); err != nil {
		return err
	}
	a.standalone = &standaloneRuntime{runtime: n}
	go n.supervise(ctx)
	go func() { <-ctx.Done(); n.sidecars.Stop() }()
	return nil
}

func (a *App) handleStandaloneImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	if a.nodeRuntime != nil || a.standalone == nil {
		writeJSON(w, 409, apiResponse{Message: "import managed nodes through control"})
		return
	}
	var input struct{ Nodes []meritNode }
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&input); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	source := r.URL.Query().Get("nodeId")
	var chosen *meritNode
	for i := range input.Nodes {
		if (source != "" && input.Nodes[i].ID == source) || (source == "" && len(input.Nodes) == 1) {
			chosen = &input.Nodes[i]
			break
		}
	}
	if chosen == nil {
		writeJSON(w, 400, apiResponse{Message: "select exactly one source node using ?nodeId=LEGACY_ID"})
		return
	}
	converted, sidecars, logs, err := convertMerit(*chosen)
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	local := a.standalone
	local.mu.Lock()
	defer local.mu.Unlock()
	m := a.manager
	m.mu.Lock()
	state := copyState(m.state)
	m.mu.Unlock()
	for _, in := range converted.Inbounds {
		duplicate := false
		for _, old := range state.Inbounds {
			if old.ID == in.ID {
				duplicate = true
				break
			}
			if old.Port == in.Port || old.Name == in.Name {
				writeJSON(w, 400, apiResponse{Message: "import conflicts with an existing inbound"})
				return
			}
		}
		if !duplicate {
			state.Inbounds = append(state.Inbounds, in)
		}
	}
	n := local.runtime
	n.mu.Lock()
	revision := uint64(1)
	existing := []fleet.Sidecar{}
	if n.applied != nil {
		revision = n.applied.Revision + 1
		existing = append(existing, n.applied.Sidecars...)
	}
	n.mu.Unlock()
	for _, s := range sidecars {
		duplicate := false
		for _, old := range existing {
			if old.ID == s.ID {
				duplicate = true
				break
			}
			if old.Port != 0 && old.Port == s.Port {
				writeJSON(w, 400, apiResponse{Message: "import conflicts with an existing sidecar port"})
				return
			}
		}
		if !duplicate {
			existing = append(existing, s)
		}
	}
	model, err := json.Marshal(state)
	var yaml []byte
	if err == nil {
		yaml, err = renderStateConfigAt(state, 0)
	}
	if err == nil {
		desired := fleet.DesiredState{NodeID: "standalone", Revision: revision, Model: model, MihomoYAML: string(yaml), Sidecars: existing, Metering: "agent"}
		desired.Digest, err = desired.ContentDigest()
		if err == nil {
			job, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
			err = n.Apply(job, desired)
			cancel()
		}
	}
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	logs = append(logs, "Traffic history and legacy subscription tokens were not imported. Existing local settings were retained.")
	writeJSON(w, 200, apiResponse{OK: true, Data: meritImportResult{Nodes: []map[string]string{{"oldNodeId": chosen.ID, "nodeId": "standalone"}}, Log: logs}})
}

func (a *App) handleStandaloneSidecars(w http.ResponseWriter, r *http.Request) {
	if a.standalone == nil {
		writeJSON(w, 409, apiResponse{Message: "managed sidecars belong to control"})
		return
	}
	n := a.standalone.runtime
	if r.Method == "GET" {
		n.mu.Lock()
		defer n.mu.Unlock()
		var configs []fleet.Sidecar
		if n.applied != nil {
			configs = n.applied.Sidecars
		}
		writeJSON(w, 200, apiResponse{OK: true, Data: configs})
		return
	}
	if r.Method != "POST" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	var configs []fleet.Sidecar
	if err := decodeJSON(r, &configs); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	a.standalone.mu.Lock()
	defer a.standalone.mu.Unlock()
	m := a.manager
	m.mu.Lock()
	state := copyState(m.state)
	m.mu.Unlock()
	n.mu.Lock()
	rev := uint64(1)
	if n.applied != nil {
		rev = n.applied.Revision + 1
	}
	n.mu.Unlock()
	yaml, err := renderStateConfigAt(state, 0)
	model, _ := json.Marshal(state)
	if err == nil {
		desired := fleet.DesiredState{NodeID: "standalone", Revision: rev, Model: model, MihomoYAML: string(yaml), Sidecars: configs, Metering: "agent"}
		desired.Digest, err = desired.ContentDigest()
		if err == nil {
			err = n.Apply(r.Context(), desired)
		}
	}
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	writeJSON(w, 200, apiResponse{OK: true})
}
