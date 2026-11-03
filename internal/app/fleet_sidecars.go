package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RomanovCaesar/m-ui/internal/fleet"
	"net"
	"strconv"
)

type fleetSidecarInput struct {
	Kind            string `json:"kind"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Domain          string `json:"domain"`
	Handshake       string `json:"handshake"`
	CertificateName string `json:"certificateName"`
}

func (f *fleetConsole) createSidecar(id string, p fleetSidecarInput) (*fleet.DesiredState, error) {
	if p.Kind != "naive" && p.Kind != "shadowtls" {
		return nil, errors.New("choose naive or shadowtls")
	}
	if p.Port < 1 || p.Port > 65535 {
		return nil, errors.New("invalid sidecar port")
	}
	ed, err := f.editor(id)
	if err != nil {
		return nil, err
	}
	ed.manager.mu.Lock()
	state := copyState(ed.manager.state)
	ed.manager.mu.Unlock()
	current := []fleet.Sidecar{}
	node := f.store.Node(id)
	if node.Desired != nil {
		current = append(current, node.Desired.Sidecars...)
	}
	used := map[int]bool{}
	for _, in := range state.Inbounds {
		used[in.Port] = true
	}
	for _, s := range current {
		used[s.Port] = true
		var cfg struct {
			Port int `json:"server_port"`
		}
		if json.Unmarshal([]byte(s.Config), &cfg) == nil {
			used[cfg.Port] = true
		}
	}
	if used[p.Port] {
		return nil, errors.New("sidecar listen port is already allocated")
	}
	if p.Username == "" {
		p.Username = "user-" + fleet.ID()[:10]
	}
	if p.Password == "" {
		p.Password = fleet.ID()
	}
	legacy := meritPort{ID: fleet.ID()[:12], Port: p.Port, ProxyType: p.Kind, Username: p.Username, Password: p.Password, Enabled: true, Domain: p.Domain}
	if p.Kind == "naive" {
		certs, err := f.certificates()
		if err != nil {
			return nil, err
		}
		c, ok := certs[p.CertificateName]
		if !ok {
			return nil, errors.New("select an imported or issued central certificate")
		}
		if err = validateFleetCertificate(&c); err != nil {
			return nil, err
		}
		if p.Domain == "" {
			return nil, errors.New("NaiveProxy requires its public TLS domain")
		}
		selected := c
		selected.Domain = p.Domain
		if err = validateFleetCertificate(&selected); err != nil {
			return nil, fmt.Errorf("certificate does not cover NaiveProxy domain: %w", err)
		}
		legacy.TLSCert = c.Certificate
		legacy.TLSKey = c.PrivateKey
	} else {
		hash := sha256.Sum256([]byte(id + legacy.ID))
		backend := 20000 + (int(hash[0])*256+int(hash[1]))%40000
		for tries := 0; used[backend] || backend == p.Port; tries++ {
			if tries > 40000 {
				return nil, errors.New("no backend port available")
			}
			backend++
			if backend > 59999 {
				backend = 20000
			}
		}
		password := make([]byte, 16)
		if _, err = rand.Read(password); err != nil {
			return nil, err
		}
		legacy.BackendManaged = true
		legacy.BackendAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(backend))
		legacy.BackendPassword = base64.StdEncoding.EncodeToString(password)
		legacy.BackendCipher = "2022-blake3-aes-128-gcm"
		legacy.HandshakeAddr = valueOr(p.Handshake, "cloudflare.com:443")
	}
	_, created, _, err := convertMerit(meritNode{ID: id, Ports: []meritPort{legacy}})
	if err != nil {
		return nil, err
	}
	desired, err := f.publish(id, ed, append(current, created...))
	if err != nil {
		return nil, err
	}
	if p.Kind == "naive" {
		certs, err := f.certificates()
		if err != nil {
			return nil, err
		}
		c := certs[p.CertificateName]
		c.Bindings = append(c.Bindings, fleetCertificateBinding{NodeID: id, SidecarID: created[0].ID})
		certs[c.Name] = c
		if err = f.saveCertificates(certs); err != nil {
			return nil, fmt.Errorf("sidecar published but certificate binding was not saved: %w", err)
		}
	}
	return desired, nil
}
