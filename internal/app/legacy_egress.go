package app

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

type meritEgress struct {
	Proxies []meritEgressProxy `json:"proxies"`
	Rules   []meritEgressRule  `json:"rules"`
}
type meritEgressProxy struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}
type meritEgressRule struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	IPRanges   []string `json:"ipRanges"`
	Domains    []string `json:"domainNames"`
	Action     string   `json:"action"`
	ProxyNames []string `json:"proxyNames"`
	Ports      []int    `json:"ports"`
	Enabled    bool     `json:"enabled"`
	Order      int      `json:"order"`
}

func convertMeritEgress(node meritNode, s *State) error {
	if node.Egress == nil {
		return nil
	}
	names := map[string]string{}
	for i, p := range node.Egress.Proxies {
		if p.Name == "" || p.Host == "" || p.Port < 1 || p.Port > 65535 || names[p.Name] != "" {
			return fmt.Errorf("invalid/duplicate legacy egress proxy %q", p.Name)
		}
		if p.Protocol != "" && p.Protocol != "SOCKS5_PROXY_PROTOCOL" {
			return fmt.Errorf("unsupported egress protocol %q", p.Protocol)
		}
		name := fmt.Sprintf("legacy-%d-%s", i, p.Name)
		names[p.Name] = name
		if p.Enabled {
			s.Outbounds = append(s.Outbounds, MihomoOutbound{ID: fmt.Sprintf("legacy-out-%d", i), Kind: "proxy", Type: "socks5", Name: name, Server: p.Host, Port: p.Port, Username: p.Username, Password: p.Password})
		}
	}
	rules := append([]meritEgressRule(nil), node.Egress.Rules...)
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Order < rules[j].Order })
	for ri, r := range rules {
		if !r.Enabled {
			continue
		}
		target := r.Action
		if r.Action == "PROXY" {
			active := []string{}
			for _, name := range r.ProxyNames {
				if names[name] == "" {
					return fmt.Errorf("egress rule references missing proxy %q", name)
				}
				for _, out := range s.Outbounds {
					if out.Name == names[name] {
						active = append(active, out.Name)
					}
				}
			}
			if len(active) == 0 {
				target = "REJECT"
			} else {
				target = active[0]
			}
			if len(active) > 1 {
				target = fmt.Sprintf("legacy-route-%d", ri)
				s.Outbounds = append(s.Outbounds, MihomoOutbound{ID: target, Kind: "group", Type: "load-balance", Strategy: "round-robin", Name: target, Proxies: active, TestURL: defaultMihomoTestURL, Interval: 300})
			}
		} else if r.Action != "DIRECT" && r.Action != "REJECT" {
			return fmt.Errorf("invalid legacy action %q", r.Action)
		}
		conditions := []string{}
		all := len(r.Domains) == 0 && len(r.IPRanges) == 0
		for _, d := range r.Domains {
			if strings.ContainsAny(d, ",()\n\r") {
				return fmt.Errorf("invalid legacy domain %q", d)
			}
			if d == "*" {
				conditions = append(conditions, "(DOMAIN-REGEX,.)")
			} else {
				conditions = append(conditions, "(DOMAIN-SUFFIX,"+d+")")
			}
		}
		for _, ip := range r.IPRanges {
			if ip == "*" {
				conditions = append(conditions, "(IP-CIDR,0.0.0.0/0,no-resolve)", "(IP-CIDR6,::/0,no-resolve)")
				continue
			}
			if parsed := net.ParseIP(ip); parsed != nil {
				if parsed.To4() != nil {
					ip += "/32"
				} else {
					ip += "/128"
				}
			}
			if _, _, err := net.ParseCIDR(ip); err != nil {
				return fmt.Errorf("invalid legacy IP range %q", ip)
			}
			kind := "IP-CIDR"
			if strings.Contains(ip, ":") {
				kind = "IP-CIDR6"
			}
			conditions = append(conditions, "("+kind+","+ip+",no-resolve)")
		}
		for _, in := range s.Inbounds {
			if in.Type != "mieru" {
				continue
			}
			selected := len(r.Ports) == 0
			for _, port := range r.Ports {
				if port < 1 || port > 65535 {
					return fmt.Errorf("invalid egress port")
				}
				if in.Port == port {
					selected = true
				}
			}
			if !selected {
				continue
			}
			scope := "(IN-NAME," + in.Name + ")"
			payload := "(" + scope + ")"
			if !all && len(conditions) > 0 {
				payload = "(" + scope + ",(OR,(" + strings.Join(conditions, ",") + ")))"
			}
			s.RoutingRules = append(s.RoutingRules, MihomoRoutingRule{ID: fmt.Sprintf("legacy-rule-%d-%s", ri, in.ID), Type: "AND", Payload: payload, Target: target})
		}
	}
	// Defaults must remain last: first-match ordering is part of the old contract.
	filtered := s.RoutingRules[:0]
	for _, r := range s.RoutingRules {
		if r.Type != "MATCH" {
			filtered = append(filtered, r)
		}
	}
	s.RoutingRules = append(filtered, defaultMihomoRoutingRules()...)
	var err error
	s.Outbounds, err = normalizeMihomoOutbounds(s.Outbounds)
	if err != nil {
		return err
	}
	s.RoutingRules, err = normalizeMihomoRoutingRules(s.RoutingRules, s.Outbounds)
	return err
}
