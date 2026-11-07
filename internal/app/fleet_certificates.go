package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
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

	"github.com/hlonglin/unifiedproxy/internal/fleet"
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

type fleetDNSCredentials struct {
	CloudflareToken string `json:"cloudflareToken,omitempty"`
}

type fleetCertificateRequest struct {
	Name           string                    `json:"name"`
	Domain         string                    `json:"domain"`
	Certificate    string                    `json:"certificate"`
	PrivateKey     string                    `json:"privateKey"`
	DNS            string                    `json:"dns"`
	Automatic      bool                      `json:"automatic"`
	ExpiresAt      time.Time                 `json:"expiresAt"`
	Bindings       []fleetCertificateBinding `json:"bindings"`
	DNSCredentials fleetDNSCredentials       `json:"dnsCredentials,omitempty"`
}

func (r fleetCertificateRequest) certificate() fleetCertificate {
	return fleetCertificate{
		Name: r.Name, Domain: r.Domain, Certificate: r.Certificate, PrivateKey: r.PrivateKey,
		DNS: r.DNS, Automatic: r.Automatic, ExpiresAt: r.ExpiresAt, Bindings: r.Bindings,
	}
}

var fleetACMEScriptURL = "https://raw.githubusercontent.com/acmesh-official/acme.sh/3.1.6/acme.sh"
var fleetACMEScriptSHA256 = "c7d68b021cfd6380ea83a82962abde5b484779fee0b97d38681dfa1396bbc8d7"

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

func validateFleetCertificateHostname(c fleetCertificate, hostname string) error {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return nil
	}
	pair, err := tls.X509KeyPair([]byte(c.Certificate), []byte(c.PrivateKey))
	if err != nil {
		return err
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return err
	}
	if err = leaf.VerifyHostname(hostname); err != nil {
		return fmt.Errorf("certificate does not cover inbound SNI %q: %w", hostname, err)
	}
	return nil
}

func fleetACMEPaths(dataDir string) (acmePath, configHome string, err error) {
	acmeRoot := filepath.Join(dataDir, "acme")
	configHome = filepath.Join(acmeRoot, "config")
	acmeHome := filepath.Join(acmeRoot, "home")
	if err = os.MkdirAll(configHome, 0700); err != nil {
		return "", "", err
	}
	if err = os.Chmod(configHome, 0700); err != nil {
		return "", "", err
	}
	if err = os.MkdirAll(acmeHome, 0700); err != nil {
		return "", "", err
	}
	if err = os.Chmod(acmeHome, 0700); err != nil {
		return "", "", err
	}
	acmePath = strings.TrimSpace(os.Getenv("UNIFIEDPROXY_ACME_PATH"))
	if acmePath != "" {
		return acmePath, configHome, nil
	}
	return filepath.Join(acmeHome, "acme.sh"), configHome, nil
}

func ensureFleetACME(ctx context.Context, dataDir string) (string, string, error) {
	acmePath, configHome, err := fleetACMEPaths(dataDir)
	if err != nil {
		return "", "", err
	}
	info, statErr := os.Stat(acmePath)
	if statErr == nil {
		if info.IsDir() || info.Mode().Perm()&0111 == 0 {
			return "", "", errors.New("configured acme.sh path is not executable")
		}
		return acmePath, configHome, nil
	}
	if !errors.Is(statErr, os.ErrNotExist) {
		return "", "", statErr
	}
	if strings.TrimSpace(os.Getenv("UNIFIEDPROXY_ACME_PATH")) != "" {
		return "", "", fmt.Errorf("acme.sh was not found at UNIFIEDPROXY_ACME_PATH %q", acmePath)
	}
	installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err = installFleetACME(installCtx, filepath.Dir(acmePath), configHome); err != nil {
		return "", "", err
	}
	info, err = os.Stat(acmePath)
	if err != nil || info.IsDir() || info.Mode().Perm()&0111 == 0 {
		if err == nil {
			err = errors.New("installer did not create an executable acme.sh")
		}
		return "", "", fmt.Errorf("acme.sh installation failed: %w", err)
	}
	return acmePath, configHome, nil
}

func verifyFleetACMEScript(source []byte) error {
	digest := sha256.Sum256(source)
	if hex.EncodeToString(digest[:]) != fleetACMEScriptSHA256 {
		return errors.New("downloaded acme.sh installer failed SHA-256 verification")
	}
	return nil
}

func installFleetACME(ctx context.Context, acmeHome, configHome string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fleetACMEScriptURL, nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: 2 * time.Minute}).Do(request)
	if err != nil {
		return fmt.Errorf("download acme.sh installer: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download acme.sh installer: HTTP %s", response.Status)
	}
	const maxInstallerSize = 8 << 20
	source, err := io.ReadAll(io.LimitReader(response.Body, maxInstallerSize+1))
	if err != nil {
		return fmt.Errorf("read acme.sh installer: %w", err)
	}
	if len(source) == 0 || len(source) > maxInstallerSize {
		return errors.New("acme.sh installer response is empty or too large")
	}
	if err = verifyFleetACMEScript(source); err != nil {
		return err
	}
	if err = os.MkdirAll(acmeHome, 0700); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "sh", "-s", "--", "--install-online", "--home", acmeHome, "--config-home", configHome, "--cert-home", configHome, "--nocron")
	cmd.Stdin = bytes.NewReader(source)
	cmd.Env = os.Environ()
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err = cmd.Run(); err != nil {
		detail := strings.TrimSpace(output.String())
		if detail == "" {
			detail = err.Error()
		}
		if len(detail) > 3000 {
			detail = detail[len(detail)-3000:]
		}
		return fmt.Errorf("install acme.sh: %s", detail)
	}
	return nil
}

func (f *fleetConsole) acquireCertificate(ctx context.Context, c fleetCertificate, renew bool, credentials fleetDNSCredentials) (fleetCertificate, error) {
	return acquireFleetCertificate(ctx, f.dataDir, c, renew, credentials)
}

func acquireFleetCertificate(ctx context.Context, dataDir string, c fleetCertificate, renew bool, credentials fleetDNSCredentials) (fleetCertificate, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	credentials.CloudflareToken = strings.TrimSpace(credentials.CloudflareToken)
	if len(credentials.CloudflareToken) > 4096 {
		return c, errors.New("Cloudflare API token is too long")
	}
	if !fleet.SafeName(c.Name) || (!fleetDomainPattern.MatchString(c.Domain) && net.ParseIP(c.Domain) == nil) {
		return c, errors.New("invalid certificate identity")
	}
	if c.DNS != "" && c.DNS != "dns_cf" {
		return c, errors.New("supported DNS challenge is dns_cf")
	}
	if c.DNS != "dns_cf" && credentials.CloudflareToken != "" {
		return c, errors.New("Cloudflare credentials require the dns_cf challenge")
	}
	if !renew && c.DNS == "dns_cf" && credentials.CloudflareToken == "" && os.Getenv("CF_Token") == "" {
		return c, errors.New("Cloudflare DNS challenge requires an API token")
	}
	if c.DNS == "" {
		if _, err := exec.LookPath("socat"); err != nil {
			return c, errors.New("HTTP ACME validation requires socat; install socat or use Cloudflare DNS")
		}
	}
	acme, configHome, err := ensureFleetACME(ctx, dataDir)
	if err != nil {
		return c, err
	}
	dir := filepath.Join(dataDir, "certificates", c.Name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return c, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return c, err
	}
	args := []string{"--issue", "--server", "letsencrypt", "-d", c.Domain, "--config-home", configHome}
	if renew {
		args[0] = "--renew"
		args = append(args, "--force")
	}
	if c.DNS == "dns_cf" {
		args = append(args, "--dns", "dns_cf")
	} else if !renew {
		args = append(args, "--standalone", "--httpport", "80")
	}
	if net.ParseIP(c.Domain) != nil {
		args = append(args, "--certificate-profile", "shortlived")
	}
	run := func(extraEnv map[string]string, args ...string) error {
		cmd := exec.CommandContext(ctx, acme, args...)
		cmd.Env = os.Environ()
		for key, value := range extraEnv {
			prefix := key + "="
			filtered := make([]string, 0, len(cmd.Env)+1)
			for _, item := range cmd.Env {
				if !strings.HasPrefix(item, prefix) {
					filtered = append(filtered, item)
				}
			}
			cmd.Env = append(filtered, prefix+value)
		}
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		if err := cmd.Run(); err != nil {
			detail := strings.TrimSpace(output.String())
			if detail == "" {
				detail = err.Error()
			}
			secret := credentials.CloudflareToken
			if secret == "" {
				secret = os.Getenv("CF_Token")
			}
			if secret != "" {
				detail = strings.ReplaceAll(detail, secret, "[redacted]")
			}
			if len(detail) > 3000 {
				detail = detail[len(detail)-3000:]
			}
			return fmt.Errorf("acme.sh %s failed: %s", args[0], detail)
		}
		return nil
	}
	issueEnv := map[string]string{}
	if credentials.CloudflareToken != "" {
		issueEnv["CF_Token"] = credentials.CloudflareToken
	}
	if err := run(issueEnv, args...); err != nil {
		return c, err
	}
	certPath, keyPath := filepath.Join(dir, "fullchain.pem"), filepath.Join(dir, "private.key")
	if err := run(nil, "--install-cert", "-d", c.Domain, "--key-file", keyPath, "--fullchain-file", certPath, "--config-home", configHome); err != nil {
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

func (f *fleetConsole) validateCertificateBindings(c fleetCertificate) error {
	for _, binding := range c.Bindings {
		if binding.SidecarID != "" {
			node := f.store.Node(binding.NodeID)
			if node == nil || node.Desired == nil {
				return errors.New("unknown sidecar certificate node")
			}
			found := false
			for _, sidecar := range node.Desired.Sidecars {
				if sidecar.ID != binding.SidecarID {
					continue
				}
				if sidecar.Kind != "naive" {
					return errors.New("only Naive sidecars accept TLS certificate bindings")
				}
				selected := c
				selected.Domain = sidecar.ServerName
				if err := validateFleetCertificate(&selected); err != nil {
					return fmt.Errorf("certificate does not cover bound Naive domain: %w", err)
				}
				found = true
			}
			if !found {
				return errors.New("unknown certificate sidecar")
			}
			continue
		}
		editor, err := f.editor(binding.NodeID)
		if err != nil {
			return err
		}
		var target Inbound
		found := false
		editor.manager.mu.Lock()
		for _, inbound := range editor.manager.state.Inbounds {
			if inbound.ID == binding.InboundID {
				target = inbound
				found = true
				break
			}
		}
		editor.manager.mu.Unlock()
		if !found {
			return errors.New("certificate binding refers to an unknown inbound")
		}
		if !inboundLoadsCertificate(target) || target.Reality.Enabled {
			return errors.New("certificate binding target does not use a standard TLS certificate")
		}
		if err = validateFleetCertificateHostname(c, target.TLSServerName); err != nil {
			return err
		}
	}
	return nil
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
				if !inboundLoadsCertificate(*in) || in.Reality.Enabled {
					m.mu.Unlock()
					return errors.New("certificate binding target does not use a standard TLS certificate")
				}
				if err := validateFleetCertificateHostname(c, in.TLSServerName); err != nil {
					m.mu.Unlock()
					return err
				}
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
	var input fleetCertificateRequest
	if err = decodeJSON(r, &input); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	c := input.certificate()
	if action != "issue" && input.DNSCredentials.CloudflareToken != "" {
		writeJSON(w, 400, apiResponse{Message: "DNS credentials are accepted only when issuing a certificate"})
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
		c, err = f.acquireCertificate(r.Context(), c, action == "renew", input.DNSCredentials)
	case "bind":
		err = validateFleetCertificate(&c)
	default:
		err = errors.New("unknown certificate operation")
	}
	if err == nil {
		err = f.validateCertificateBindings(c)
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
			updated, err := f.acquireCertificate(job, c, true, fleetDNSCredentials{})
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
