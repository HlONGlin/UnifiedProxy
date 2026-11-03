package app

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
)

type fleetSubscriptionSidecar struct {
	NodeName string
	Server   string
	Config   fleet.Sidecar
}

func (a *App) refreshFleetSubscriptions() error {
	if a.fleet == nil {
		configs := a.localSidecarConfigs()
		if len(configs) == 0 {
			return nil
		}
		m := a.manager
		m.mu.Lock()
		defer m.mu.Unlock()
		clients := allStateClients(m.state.Inbounds)
		for _, s := range configs {
			if s.Enabled && s.Username != "" {
				clients = append(clients, Client{Username: s.Username})
			}
		}
		before := len(m.state.SubscriptionTokens)
		if err := m.ensureSubscriptionTokensLocked(clients); err != nil {
			return err
		}
		if len(m.state.SubscriptionTokens) != before {
			return m.saveLocked()
		}
		return nil
	}
	clients := []Client{}
	for _, node := range a.fleet.store.Nodes() {
		if node.Revoked || node.Desired == nil {
			continue
		}
		var state State
		if err := json.Unmarshal(node.Desired.Model, &state); err != nil {
			return err
		}
		clients = append(clients, allStateClients(state.Inbounds)...)
		for _, s := range node.Desired.Sidecars {
			if s.Enabled && s.Username != "" {
				clients = append(clients, Client{Username: s.Username})
			}
		}
	}
	m := a.manager
	m.mu.Lock()
	defer m.mu.Unlock()
	before := len(m.state.SubscriptionTokens)
	if err := m.ensureSubscriptionTokensLocked(clients); err != nil {
		return err
	}
	if len(m.state.SubscriptionTokens) != before {
		return m.saveLocked()
	}
	return nil
}

func (a *App) fleetSubscriptionSnapshot(path string) (subscriptionSnapshot, subscriptionFormat, bool) {
	if err := a.refreshFleetSubscriptions(); err != nil {
		return subscriptionSnapshot{}, subscriptionFormatPage, false
	}
	m := a.manager
	m.mu.Lock()
	settings := m.state.Settings
	token, format, ok := matchSubscriptionPath(settings, path)
	if !ok {
		m.mu.Unlock()
		return subscriptionSnapshot{}, subscriptionFormatPage, false
	}
	username := ""
	for user, t := range m.state.SubscriptionTokens {
		if t == token {
			username = user
			break
		}
	}
	m.mu.Unlock()
	if username == "" {
		return subscriptionSnapshot{}, subscriptionFormatPage, false
	}
	snapshot := subscriptionSnapshot{Settings: settings, Username: username, Token: token}
	for _, node := range a.fleet.store.Nodes() {
		if node.Revoked || node.Desired == nil || node.Address == "" {
			continue
		}
		var state State
		if json.Unmarshal(node.Desired.Model, &state) != nil {
			continue
		}
		for _, in := range state.Inbounds {
			in.Name = node.Name + " / " + in.Name
			for _, client := range in.Clients {
				if subscriptionUsername(client) != username {
					continue
				}
				if usage, ok := node.Status.Traffic[in.ID+":"+client.ID]; ok {
					client.Traffic = TrafficStats{Up: usage.Up, Down: usage.Down, AllTimeUp: usage.AllTimeUp, AllTimeDown: usage.AllTimeDown}
					client.LastOnline = usage.LastOnline
					if node.Status.Revision == node.Desired.Revision && node.Status.Digest == node.Desired.Digest && usage.ExpiryTime > client.ExpiryTime {
						client.ExpiryTime = usage.ExpiryTime
					}
				}
				if usage, ok := node.Status.Traffic[in.ID]; ok {
					in.Traffic = TrafficStats{Up: usage.Up, Down: usage.Down, AllTimeUp: usage.AllTimeUp, AllTimeDown: usage.AllTimeDown}
				}
				snapshot.Clients = append(snapshot.Clients, subscriptionClient{Inbound: in, Client: client, Server: node.Address})
			}
		}
		for _, s := range node.Desired.Sidecars {
			if s.Enabled && s.Username == username {
				snapshot.FleetSidecars = append(snapshot.FleetSidecars, fleetSubscriptionSidecar{NodeName: node.Name, Server: node.Address, Config: s})
			}
		}
	}
	return snapshot, format, len(snapshot.Clients) > 0 || len(snapshot.FleetSidecars) > 0
}

func (a *App) localSidecarConfigs() []fleet.Sidecar {
	n := a.nodeRuntime
	if n == nil && a.standalone != nil {
		n = a.standalone.runtime
	}
	if n == nil {
		return nil
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.applied == nil {
		return nil
	}
	return append([]fleet.Sidecar(nil), n.applied.Sidecars...)
}

func fleetSidecarLinks(snapshot subscriptionSnapshot, defaultServer string) []subscriptionNode {
	nodes := []subscriptionNode{}
	for _, item := range snapshot.FleetSidecars {
		s := item.Config
		name := item.NodeName + " / " + s.ID
		host := valueOr(s.Address, valueOr(item.Server, defaultServer))
		if s.Port < 1 {
			continue
		}
		switch s.Kind {
		case "naive":
			u := url.URL{Scheme: "naive+https", Host: net.JoinHostPort(host, strconv.Itoa(s.Port)), User: url.UserPassword(s.Username, s.Password), Fragment: name}
			nodes = append(nodes, subscriptionNode{Name: name, URL: u.String()})
		case "shadowtls":
			u := url.URL{Scheme: "ss", Host: net.JoinHostPort(host, strconv.Itoa(s.Port)), User: url.User(base64.RawURLEncoding.EncodeToString([]byte(s.Cipher + ":" + s.BackendPassword))), Fragment: name}
			query := url.Values{}
			query.Set("plugin", "shadow-tls;host="+s.ServerName+";password="+s.Password+";version=3")
			u.RawQuery = query.Encode()
			nodes = append(nodes, subscriptionNode{Name: name, URL: u.String()})
		}
	}
	return nodes
}
func fleetSidecarProxies(snapshot subscriptionSnapshot, defaultServer string) []map[string]any {
	proxies := []map[string]any{}
	for _, item := range snapshot.FleetSidecars {
		s := item.Config
		if s.Kind != "shadowtls" || s.Port < 1 {
			continue
		}
		proxies = append(proxies, map[string]any{"name": item.NodeName + " / " + s.ID, "type": "ss", "server": valueOr(s.Address, valueOr(item.Server, defaultServer)), "port": s.Port, "cipher": s.Cipher, "password": s.BackendPassword, "plugin": "shadow-tls", "plugin-opts": map[string]any{"host": s.ServerName, "password": s.Password, "version": 3}})
	}
	return proxies
}
