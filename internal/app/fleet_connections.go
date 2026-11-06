package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Reloading a listener only prevents new connections. Existing sessions must
// be closed separately when a policy disables their inbound or user.
func restrictedSessions(before, after State, now int64) map[string]map[string]bool {
	inbounds := make(map[string]Inbound, len(after.Inbounds))
	for _, in := range after.Inbounds {
		inbounds[in.ID] = in
	}
	blocked := map[string]map[string]bool{}
	for _, old := range before.Inbounds {
		if inboundDepleted(old, now) {
			continue
		}
		in, exists := inbounds[old.ID]
		if !exists || inboundDepleted(in, now) || old.Name != in.Name {
			blocked[old.Name] = nil
			continue
		}
		active := map[string]bool{}
		for _, c := range in.Clients {
			if !clientDepleted(c, now) {
				active[c.ID] = true
			}
		}
		users := map[string]bool{}
		for _, c := range old.Clients {
			if active[c.ID] || clientDepleted(c, now) {
				continue
			}
			for _, name := range []string{c.Name, c.Username, c.UUID} {
				if name != "" {
					users[name] = true
				}
			}
		}
		if len(users) > 0 {
			blocked[old.Name] = users
		}
	}
	return blocked
}

func (n *nodeRuntime) closeRestrictedSessions(ctx context.Context, before, after State) error {
	blocked := restrictedSessions(before, after, time.Now().UnixMilli())
	if len(blocked) == 0 {
		return nil
	}
	snapshot, err := n.manager.coreGet("/connections")
	if err != nil {
		return fmt.Errorf("read sessions for disabled inbound: %w", err)
	}
	root, _ := snapshot.(map[string]any)
	connections, _ := root["connections"].([]any)
	for _, raw := range connections {
		connection, _ := raw.(map[string]any)
		metadata, _ := connection["metadata"].(map[string]any)
		name, _ := metadata["inboundName"].(string)
		user, _ := metadata["inboundUser"].(string)
		users, restricted := blocked[name]
		if !restricted || (users != nil && !users[user]) {
			continue
		}
		id, _ := connection["id"].(string)
		if id == "" {
			continue
		}
		if err := n.coreHTTP(ctx, http.MethodDelete, "/connections/"+url.PathEscape(id), nil); err != nil {
			return fmt.Errorf("close session for disabled inbound: %w", err)
		}
	}
	return nil
}
