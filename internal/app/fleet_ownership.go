package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func (a *App) localUnmanagedMutation(r *http.Request) bool {
	n := a.nodeRuntime
	if n == nil {
		return true
	}
	if r.URL.Path != "/api/inbounds" && r.URL.Path != "/api/clients" {
		return false
	}
	var managed State
	if n.applied == nil {
		return false
	}
	err := json.Unmarshal(n.applied.Model, &managed)
	if err != nil {
		return false
	}
	id := r.URL.Query().Get("id")
	if r.URL.Path == "/api/clients" {
		id = r.URL.Query().Get("inboundId")
	}
	if r.Method != "DELETE" {
		data, err := io.ReadAll(io.LimitReader(r.Body, (2<<20)+1))
		r.Body = io.NopCloser(bytes.NewReader(data))
		if err != nil || len(data) > 2<<20 {
			return false
		}
		var p struct {
			ID        string
			InboundID string
		}
		if json.Unmarshal(data, &p) != nil {
			return false
		}
		id = p.ID
		if r.URL.Path == "/api/clients" {
			id = p.InboundID
		}
	}
	if r.URL.Path == "/api/clients" && id == "" {
		return false
	}
	for _, in := range managed.Inbounds {
		if in.ID == id {
			return false
		}
	}
	a.manager.mu.Lock()
	applying := a.manager.fleetApplying
	a.manager.mu.Unlock()
	return !applying
}
