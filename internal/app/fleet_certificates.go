package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
)

type fleetCertificateBinding struct {
	NodeID    string `json:"nodeId"`
	InboundID string `json:"inboundId"`
	SidecarID string `json:"sidecarId,omitempty"`
}
type fleetCertificate struct {
	Name        string                    `json:"name"`
	Domain      string                    `json:"domain"`
	Certificate string                    `json:"certificate"`
	PrivateKey  string                    `json:"privateKey"`
	DNS         string                    `json:"dns,omitempty"`
	Automatic   bool                      `json:"automatic"`
	ExpiresAt   time.Time                 `json:"expiresAt"`
	Bindings    []fleetCertificateBinding `json:"bindings,omitempty"`
	LastError   string                    `json:"lastError,omitempty"`
}

const fleetCertificateFile = "fleet-certificates.json"

var fleetDomainPattern = regexp.MustCompile(`^(\*\.)?[A-Za-z0-9][A-Za-z0-9.-]{0,250}[A-Za-z0-9]$`)

func (f *fleetConsole) certificates() (map[string]fleetCertificate, error) {
	certs := map[string]fleetCertificate{}
	err := fleet.ReadJSON(filepath.Join(f.dataDir, fleetCertificateFile), &certs)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return certs, err
}
func (f *fleetConsole) saveCertificates(certs map[string]fleetCertificate) error {
	return fleet.WriteJSON(filepath.Join(f.dataDir, fleetCertificateFile), certs)
}
func validateFleetCertificate(c *fleetCertificate) error {
	if !fleet.SafeName(c.Name) {
		return errors.New("invalid certificate name")
	}
	pair, err := tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey))
	if err != nil {
		return err
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return err
	}
	if !time.Now().Before(leaf.NotAfter) {
		return errors.New("certificate has expired")
	}
	if time.Now().Before(leaf.NotBefore) {
		return errors.New("certificate is not valid yet")
	}
	if c.Domain != "" {
		domain := strings.Replace(c.Domain, "*.", "probe.", 1)
		if err = leaf.VerifyHostname(domain); err != nil {
			return err
		}
	}
	c.ExpiresAt = leaf.NotAfter
	return nil
}

func (f *fleetConsole) acquireCertificate(ctx context.Context, c fleetCertificate, renew bool) (fleetCertificate, error) {
	if !fleet.SafeName(c.Name) || (!fleetDomainPattern.MatchString(c.Domain) && net.ParseIP(c.Domain) == nil) {
		return c, errors.New("invalid certificate identity")
	}
	if c.DNS != "" && c.DNS != "dns_cf" {
		return c, errors.New("supported DNS challenge is dns_cf")
	}
	acme := os.Getenv("MUI_ACME_PATH")
	if acme == "" {
		acme = "/root/.acme.sh/acme.sh"
	}
	dir := filepath.Join(f.dataDir, "certificates", c.Name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return c, err
	}
	args := []string{"--issue", "--server", "letsencrypt", "-d", c.Domain}
	if renew {
		args[0] = "--renew"
	}
	if c.DNS == "dns_cf" {
		args = append(args, "--dns", "dns_cf")
	} else if !renew {
		args = append(args, "--standalone", "--httpport", "80")
	}
	if net.ParseIP(c.Domain) != nil {
		args = append(args, "--certificate-profile", "shortlived")
	}
	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, acme, args...)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("acme.sh failed (check local ACME logs): %w", err)
		}
		return nil
	}
	if err := run(args...); err != nil {
		return c, err
	}
	certPath, keyPath := filepath.Join(dir, "fullchain.pem"), filepath.Join(dir, "private.key")
	if err := run("--install-cert", "-d", c.Domain, "--key-file", keyPath, "--fullchain-file", certPath); err != nil {
		return c, err
	}
	pair, _, _, err := loadPanelCertificate(certPath, keyPath)
	if err != nil {
		return c, err
	}
	var chain []byte
	for _, der := range pair.Certificate {
		chain = append(chain, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return c, err
	}
	c.Certificate = string(chain)
	c.PrivateKey = string(key)
	c.LastError = ""
	if err = validateFleetCertificate(&c); err != nil {
		return c, err
	}
	return c, nil
}

func (f *fleetConsole) distributeCertificate(c fleetCertificate) error {
	for _, binding := range c.Bindings {
		if binding.SidecarID != "" {
			node := f.store.Node(binding.NodeID)
			if node == nil || node.Desired == nil {
				return errors.New("unknown sidecar certificate node")
			}
			sidecars := append([]fleet.Sidecar(nil), node.Desired.Sidecars...)
			found := false
			for i := range sidecars {
				if sidecars[i].ID != binding.SidecarID {
					continue
				}
				if sidecars[i].Kind != "naive" {
					return errors.New("only Naive sidecars accept TLS certificate bindings")
				}
				selected := c
				selected.Domain = sidecars[i].ServerName
				if err := validateFleetCertificate(&selected); err != nil {
					return fmt.Errorf("renewed certificate does not cover bound Naive domain: %w", err)
				}
				var cfg map[string]any
				if err := json.Unmarshal([]byte(sidecars[i].Config), &cfg); err != nil {
					return err
				}
				apps, ok := cfg["apps"].(map[string]any)
				if !ok {
					return errors.New("invalid Caddy apps")
				}
				apps["tls"] = map[string]any{"certificates": map[string]any{"load_pem": []any{map[string]string{"certificate": c.Certificate, "key": c.PrivateKey}}}}
				raw, _ := json.Marshal(cfg)
				sidecars[i].Config = string(raw)
				found = true
			}
			if !found {
				return errors.New("unknown certificate sidecar")
			}
			ed, err := f.editor(binding.NodeID)
			if err != nil {
				return err
			}
			if _, err = f.publish(binding.NodeID, ed, sidecars); err != nil {
				return err
			}
			continue
		}
		editor, err := f.editor(binding.NodeID)
		if err != nil {
			return err
		}
		m := editor.manager
		m.mu.Lock()
		found := false
		for i := range m.state.Inbounds {
			in := &m.state.Inbounds[i]
			if in.ID == binding.InboundID {
				in.Certificate = c.Certificate
				in.PrivateKey = c.PrivateKey
				in.CertificateMode = "content"
				found = true
			}
		}
		if !found {
			m.mu.Unlock()
			return errors.New("certificate binding refers to an unknown inbound")
		}
		err = m.saveLocked()
		m.mu.Unlock()
		if err != nil {
			return err
		}
		if _, err = f.publish(binding.NodeID, editor, nil); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) handleFleetCertificates(w http.ResponseWriter, r *http.Request, action string) {
	f := a.fleet
	f.mu.Lock()
	defer f.mu.Unlock()
	certs, err := f.certificates()
	if err != nil {
		writeJSON(w, 500, apiResponse{Message: err.Error()})
		return
	}
	if r.Method == "GET" && action == "" {
		views := []fleetCertificate{}
		for _, c := range certs {
			c.Certificate = ""
			c.PrivateKey = ""
			views = append(views, c)
		}
		writeJSON(w, 200, apiResponse{OK: true, Data: views})
		return
	}
	if r.Method != "POST" {
		writeJSON(w, 405, apiResponse{})
		return
	}
	var c fleetCertificate
	if err = decodeJSON(r, &c); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	if !fleet.SafeName(c.Name) {
		writeJSON(w, 400, apiResponse{Message: "invalid certificate name"})
		return
	}
	if previous, ok := certs[c.Name]; ok {
		if c.Bindings == nil {
			c.Bindings = previous.Bindings
		}
		if action == "renew" {
			c = previous
		}
		if action == "bind" {
			previous.Bindings = c.Bindings
			c = previous
		}
	}
	switch action {
	case "import":
		err = validateFleetCertificate(&c)
	case "issue", "renew":
		c, err = f.acquireCertificate(r.Context(), c, action == "renew")
	case "bind":
		err = validateFleetCertificate(&c)
	default:
		err = errors.New("unknown certificate operation")
	}
	if err == nil {
		certs[c.Name] = c
		err = f.saveCertificates(certs)
	}
	if err == nil {
		err = f.distributeCertificate(c)
	}
	if err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	c.Certificate = ""
	c.PrivateKey = ""
	writeJSON(w, 200, apiResponse{OK: true, Data: c})
}

func (a *App) renewFleetCertificates(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		f := a.fleet
		f.mu.Lock()
		certs, err := f.certificates()
		if err != nil {
			f.mu.Unlock()
			continue
		}
		for name, c := range certs {
			threshold := 30 * 24 * time.Hour
			if net.ParseIP(c.Domain) != nil {
				threshold = 24 * time.Hour
			}
			if !c.Automatic || time.Until(c.ExpiresAt) > threshold {
				continue
			}
			job, cancel := context.WithTimeout(ctx, 10*time.Minute)
			updated, err := f.acquireCertificate(job, c, true)
			cancel()
			if err == nil {
				err = f.distributeCertificate(updated)
			}
			if err != nil {
				c.LastError = err.Error()
				certs[name] = c
			} else {
				certs[name] = updated
			}
		}
		_ = f.saveCertificates(certs)
		f.mu.Unlock()
	}
}
