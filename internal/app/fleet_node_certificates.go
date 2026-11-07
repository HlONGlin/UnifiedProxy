package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

const (
	nodeCertificateReferencePrefix = "__NODE_CERT_DIR__/"
	nodeCertificateRegistryFile    = "managed-certificates.json"
)

type nodeCertificateIssueRequest struct {
	Name          string `json:"name"`
	Domain        string `json:"domain"`
	DNS           string `json:"dns,omitempty"`
	Automatic     bool   `json:"automatic"`
	InboundID     string `json:"inboundId"`
	TLSServerName string `json:"tlsServerName,omitempty"`
}

type nodeManagedCertificate struct {
	Name       string    `json:"name"`
	Domain     string    `json:"domain"`
	DNS        string    `json:"dns,omitempty"`
	Automatic  bool      `json:"automatic"`
	ExpiresAt  time.Time `json:"expiresAt"`
	InboundIDs []string  `json:"inboundIds,omitempty"`
	LastError  string    `json:"lastError,omitempty"`
}

func nodeCertificateReference(name, extension string) string {
	return nodeCertificateReferencePrefix + name + extension
}

func nodeCertificateReferenceName(in Inbound) (string, bool, error) {
	certRef := strings.HasPrefix(in.Certificate, nodeCertificateReferencePrefix)
	keyRef := strings.HasPrefix(in.PrivateKey, nodeCertificateReferencePrefix)
	if !certRef && !keyRef {
		return "", false, nil
	}
	if !certRef || !keyRef {
		return "", false, errors.New("incomplete node-local certificate reference")
	}
	certName := strings.TrimPrefix(in.Certificate, nodeCertificateReferencePrefix)
	keyName := strings.TrimPrefix(in.PrivateKey, nodeCertificateReferencePrefix)
	if !strings.HasSuffix(certName, ".crt") || !strings.HasSuffix(keyName, ".key") {
		return "", false, errors.New("invalid node-local certificate reference")
	}
	name := strings.TrimSuffix(certName, ".crt")
	if name == "" || !fleet.SafeName(name) || keyName != name+".key" {
		return "", false, errors.New("invalid node-local certificate reference")
	}
	return name, true, nil
}

func nodeCertificatePaths(dataDir, name string) (string, string, error) {
	if !fleet.SafeName(name) {
		return "", "", errors.New("invalid certificate name")
	}
	dir := filepath.Join(dataDir, "certificates", name)
	return filepath.Join(dir, "fullchain.pem"), filepath.Join(dir, "private.key"), nil
}

func resolveNodeCertificateReference(dataDir string, in *Inbound) error {
	name, isReference, err := nodeCertificateReferenceName(*in)
	if err != nil || !isReference {
		return err
	}
	certPath, keyPath, err := nodeCertificatePaths(dataDir, name)
	if err != nil {
		return err
	}
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("node-local certificate %q is unavailable: %w", name, err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("node-local private key %q is unavailable: %w", name, err)
	}
	c := fleetCertificate{Name: name, Certificate: string(certPEM), PrivateKey: string(keyPEM)}
	if err = validateFleetCertificate(&c); err != nil {
		return fmt.Errorf("invalid node-local certificate %q: %w", name, err)
	}
	if err = validateFleetCertificateHostname(c, in.TLSServerName); err != nil {
		return err
	}
	in.Certificate = filepath.ToSlash(certPath)
	in.PrivateKey = filepath.ToSlash(keyPath)
	in.CertificateMode = "file"
	return nil
}

func nodeManagedCertificatesPath(dataDir string) string {
	return filepath.Join(dataDir, "acme", nodeCertificateRegistryFile)
}

func readNodeManagedCertificates(dataDir string) (map[string]nodeManagedCertificate, error) {
	certificates := map[string]nodeManagedCertificate{}
	err := fleet.ReadJSON(nodeManagedCertificatesPath(dataDir), &certificates)
	if errors.Is(err, os.ErrNotExist) {
		return certificates, nil
	}
	return certificates, err
}

func saveNodeManagedCertificates(dataDir string, certificates map[string]nodeManagedCertificate) error {
	dir := filepath.Join(dataDir, "acme")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	return fleet.WriteJSON(nodeManagedCertificatesPath(dataDir), certificates)
}

func certificateDomainCovers(domain, hostname string) bool {
	domain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
	hostname = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
	if domain == hostname {
		return true
	}
	if !strings.HasPrefix(domain, "*.") {
		return false
	}
	suffix := strings.TrimPrefix(domain, "*")
	if !strings.HasSuffix(hostname, suffix) {
		return false
	}
	label := strings.TrimSuffix(hostname, suffix)
	return label != "" && !strings.Contains(label, ".")
}

func (n *nodeRuntime) issueNodeCertificate(ctx context.Context, request nodeCertificateIssueRequest) (json.RawMessage, error) {
	if !fleet.SafeName(request.Name) || (!fleetDomainPattern.MatchString(request.Domain) && net.ParseIP(request.Domain) == nil) {
		return nil, errors.New("invalid certificate identity")
	}
	if request.DNS != "" && request.DNS != "dns_cf" {
		return nil, errors.New("supported DNS challenge is dns_cf")
	}
	if strings.HasPrefix(request.Domain, "*.") && request.DNS != "dns_cf" {
		return nil, errors.New("wildcard certificates require Cloudflare DNS validation")
	}
	if request.InboundID == "" {
		return nil, errors.New("a TLS inbound is required")
	}

	if request.TLSServerName != "" && !certificateDomainCovers(request.Domain, request.TLSServerName) {
		return nil, fmt.Errorf("certificate domain %q does not cover inbound SNI %q", request.Domain, request.TLSServerName)
	}

	n.certificateMu.Lock()
	defer n.certificateMu.Unlock()

	certificates, err := readNodeManagedCertificates(n.manager.dataDir)
	if err != nil {
		return nil, err
	}
	if existing, ok := certificates[request.Name]; ok {
		existingDomain := strings.TrimSuffix(strings.TrimSpace(existing.Domain), ".")
		requestedDomain := strings.TrimSuffix(strings.TrimSpace(request.Domain), ".")
		if !strings.EqualFold(existingDomain, requestedDomain) {
			return nil, errors.New("certificate name is already used for another domain on this node")
		}
	}

	issued, err := acquireFleetCertificate(ctx, n.manager.dataDir, fleetCertificate{
		Name: request.Name, Domain: request.Domain, DNS: request.DNS, Automatic: request.Automatic,
	}, false, fleetDNSCredentials{})
	if err != nil {
		return nil, err
	}
	if err = validateFleetCertificateHostname(issued, request.TLSServerName); err != nil {
		return nil, err
	}

	metadata := certificates[request.Name]
	metadata.Name = request.Name
	metadata.Domain = request.Domain
	metadata.DNS = request.DNS
	metadata.Automatic = request.Automatic
	metadata.ExpiresAt = issued.ExpiresAt
	metadata.LastError = ""
	if !containsString(metadata.InboundIDs, request.InboundID) {
		metadata.InboundIDs = append(metadata.InboundIDs, request.InboundID)
	}
	certificates[request.Name] = metadata
	if err = saveNodeManagedCertificates(n.manager.dataDir, certificates); err != nil {
		return nil, err
	}
	return json.Marshal(metadata)
}

func (n *nodeRuntime) managedCertificates() []nodeManagedCertificate {
	n.certificateMu.Lock()
	defer n.certificateMu.Unlock()
	certificates, err := readNodeManagedCertificates(n.manager.dataDir)
	if err != nil {
		return nil
	}
	result := make([]nodeManagedCertificate, 0, len(certificates))
	for _, certificate := range certificates {
		result = append(result, certificate)
	}
	return result
}

func (n *nodeRuntime) renewNodeCertificate(ctx context.Context, name string) (json.RawMessage, error) {
	n.certificateMu.Lock()
	defer n.certificateMu.Unlock()
	certificates, err := readNodeManagedCertificates(n.manager.dataDir)
	if err != nil {
		return nil, err
	}
	metadata, ok := certificates[name]
	if !ok {
		return nil, errors.New("unknown node-local certificate")
	}
	updated, err := acquireFleetCertificate(ctx, n.manager.dataDir, fleetCertificate{
		Name: metadata.Name, Domain: metadata.Domain, DNS: metadata.DNS, Automatic: metadata.Automatic,
	}, true, fleetDNSCredentials{})
	if err != nil {
		metadata.LastError = err.Error()
		certificates[name] = metadata
		_ = saveNodeManagedCertificates(n.manager.dataDir, certificates)
		return nil, err
	}
	metadata.ExpiresAt = updated.ExpiresAt
	metadata.LastError = ""
	certificates[name] = metadata
	if err = saveNodeManagedCertificates(n.manager.dataDir, certificates); err != nil {
		return nil, err
	}
	if err = n.reloadNodeCertificateIfUsed(ctx, metadata); err != nil {
		metadata.LastError = err.Error()
		certificates[name] = metadata
		_ = saveNodeManagedCertificates(n.manager.dataDir, certificates)
		return nil, err
	}
	return json.Marshal(metadata)
}

func (n *nodeRuntime) reloadNodeCertificateIfUsed(ctx context.Context, certificate nodeManagedCertificate) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	m := n.manager
	m.mu.Lock()
	running := m.runningLocked()
	state := copyState(m.state)
	m.mu.Unlock()
	if !running {
		return nil
	}
	certPath, _, err := nodeCertificatePaths(m.dataDir, certificate.Name)
	if err != nil {
		return err
	}
	used := false
	for _, inbound := range state.Inbounds {
		if containsString(certificate.InboundIDs, inbound.ID) && filepath.Clean(inbound.Certificate) == filepath.Clean(certPath) {
			used = true
			break
		}
	}
	if !used {
		return nil
	}
	if err = m.writeConfig(); err != nil {
		return err
	}
	return n.reload(ctx, filepath.Join(m.dataDir, "config.yaml"))
}

func (n *nodeRuntime) renewManagedCertificates(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		certificates := n.managedCertificates()
		for _, certificate := range certificates {
			threshold := 30 * 24 * time.Hour
			if net.ParseIP(certificate.Domain) != nil {
				threshold = 24 * time.Hour
			}
			if !certificate.Automatic || time.Until(certificate.ExpiresAt) > threshold {
				continue
			}
			job, cancel := context.WithTimeout(ctx, 10*time.Minute)
			_, _ = n.renewNodeCertificate(job, certificate.Name)
			cancel()
		}
	}
}

func (a *App) handleFleetNodeCertificateIssue(w http.ResponseWriter, r *http.Request, nodeID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiResponse{})
		return
	}
	var request nodeCertificateIssueRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: err.Error()})
		return
	}
	if !fleet.SafeName(request.Name) || (!fleetDomainPattern.MatchString(request.Domain) && net.ParseIP(request.Domain) == nil) {
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: "invalid certificate identity"})
		return
	}
	if request.DNS != "" && request.DNS != "dns_cf" {
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: "supported DNS challenge is dns_cf"})
		return
	}
	if strings.HasPrefix(request.Domain, "*.") && request.DNS != "dns_cf" {
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: "wildcard certificates require Cloudflare DNS validation"})
		return
	}
	if request.InboundID == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: "a TLS inbound is required"})
		return
	}

	f := a.fleet
	f.mu.Lock()
	node := f.store.Node(nodeID)
	if node == nil || node.Revoked {
		f.mu.Unlock()
		writeJSON(w, http.StatusNotFound, apiResponse{Message: "unknown or revoked child server"})
		return
	}
	if !fleet.Supports(node.Status.Capabilities, "cert-issue") {
		f.mu.Unlock()
		writeJSON(w, http.StatusConflict, apiResponse{Message: "upgrade this child server to enable local certificate management"})
		return
	}
	editor, err := f.editor(nodeID)
	if err != nil {
		f.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: err.Error()})
		return
	}
	editor.manager.mu.Lock()
	previous := copyState(editor.manager.state)
	inboundIndex := -1
	for i := range editor.manager.state.Inbounds {
		inbound := editor.manager.state.Inbounds[i]
		if inbound.ID == request.InboundID {
			inboundIndex = i
			if !inboundLoadsCertificate(inbound) || inbound.Reality.Enabled {
				err = errors.New("target is not a standard TLS inbound")
			} else if inbound.TLSServerName != "" && !certificateDomainCovers(request.Domain, inbound.TLSServerName) {
				err = fmt.Errorf("certificate domain %q does not cover inbound SNI %q", request.Domain, inbound.TLSServerName)
			}
			break
		}
	}
	if inboundIndex < 0 && err == nil {
		err = errors.New("unknown TLS inbound")
	}
	if err == nil {
		inbound := &editor.manager.state.Inbounds[inboundIndex]
		request.TLSServerName = inbound.TLSServerName
		inbound.Certificate = nodeCertificateReference(request.Name, ".crt")
		inbound.PrivateKey = nodeCertificateReference(request.Name, ".key")
		inbound.CertificateMode = "file"
		err = editor.manager.saveLocked()
	}
	if err != nil {
		editor.manager.mu.Unlock()
		f.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: err.Error()})
		return
	}
	editor.manager.mu.Unlock()

	payload, err := json.Marshal(request)
	if err != nil {
		editor.manager.mu.Lock()
		editor.manager.state = previous
		_ = editor.manager.saveLocked()
		editor.manager.mu.Unlock()
		f.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: err.Error()})
		return
	}
	task, err := f.store.Enqueue(nodeID, "cert-issue", fleet.ID(), payload, 20*time.Minute)
	if err != nil {
		editor.manager.mu.Lock()
		editor.manager.state = previous
		_ = editor.manager.saveLocked()
		editor.manager.mu.Unlock()
		f.mu.Unlock()
		writeJSON(w, http.StatusBadRequest, apiResponse{Message: err.Error()})
		return
	}
	desired, publishErr := f.publish(nodeID, editor, nil)
	f.mu.Unlock()
	if publishErr != nil {
		writeJSON(w, http.StatusAccepted, apiResponse{OK: true, Data: map[string]any{"task": task, "publishError": publishErr.Error()}})
		return
	}
	if err = a.refreshFleetSubscriptions(); err != nil {
		writeJSON(w, http.StatusAccepted, apiResponse{OK: true, Data: map[string]any{"task": task, "revision": desired.Revision, "warning": err.Error()}})
		return
	}
	writeJSON(w, http.StatusAccepted, apiResponse{OK: true, Data: map[string]any{"task": task, "revision": desired.Revision}})
}
