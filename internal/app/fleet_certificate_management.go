package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type nodeDNSCredentialRequest struct {
	fleetDNSCredentials
	Name   string `json:"name"`
	Action string `json:"action,omitempty"`
}

func (n *nodeRuntime) manageDNSCredentials(request nodeDNSCredentialRequest) (json.RawMessage, error) {
	name := nodeCredentialName(request.Name)
	if !fleet.SafeName(name) {
		return nil, errors.New("invalid credential name")
	}
	n.certificateMu.Lock()
	defer n.certificateMu.Unlock()
	profiles, err := readNodeDNSProfiles(n.manager.dataDir)
	if err != nil {
		return nil, err
	}
	switch request.Action {
	case "delete":
		certificates, err := readNodeManagedCertificates(n.manager.dataDir)
		if err != nil {
			return nil, err
		}
		for _, certificate := range certificates {
			if certificate.DNS == "dns_cf" && nodeCredentialName(certificate.CredentialName) == name {
				return nil, errors.New("Cloudflare credentials are used by a managed certificate; change its credentials or delete the certificate first")
			}
		}
		delete(profiles, name)
	case "", "save":
		credentials, err := normalizeFleetDNSCredentials(request.fleetDNSCredentials)
		if err != nil {
			return nil, err
		}
		if credentials.CloudflareToken == "" {
			return nil, errors.New("enter a Cloudflare API token")
		}
		profiles[name] = credentials
	default:
		return nil, errors.New("unknown credential operation")
	}
	// Drop cached copies as well, so updating/deleting a profile cannot leave old
	// account IDs or tokens in ACME's fallback configuration.
	certificates, err := readNodeManagedCertificates(n.manager.dataDir)
	if err != nil {
		return nil, err
	}
	for _, certificate := range certificates {
		if certificate.DNS == "dns_cf" && nodeCredentialName(certificate.CredentialName) == name {
			if err = clearFleetCloudflareCache(filepath.Join(n.manager.dataDir, "certificates", certificate.Name, "acme")); err != nil {
				return nil, err
			}
		}
	}
	if name == "default" {
		if err = clearFleetCloudflareCache(filepath.Join(n.manager.dataDir, "acme", "config")); err != nil {
			return nil, err
		}
	}
	if err = saveNodeDNSProfiles(n.manager.dataDir, profiles); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"name": name, "deleted": request.Action == "delete", "cloudflareCredentialsConfigured": len(profiles) > 0})
}

type nodeCertificateManageRequest struct {
	Name           string `json:"name"`
	Automatic      bool   `json:"automatic"`
	CredentialName string `json:"credentialName,omitempty"`
}

func stateUsesNodeCertificate(state State, dataDir, name string) bool {
	certPath, keyPath, err := nodeCertificatePaths(dataDir, name)
	if err != nil {
		return false
	}
	for _, in := range state.Inbounds {
		if in.Certificate == nodeCertificateReference(name, ".crt") || in.PrivateKey == nodeCertificateReference(name, ".key") ||
			(in.Certificate != "" && filepath.Clean(in.Certificate) == filepath.Clean(certPath)) ||
			(in.PrivateKey != "" && filepath.Clean(in.PrivateKey) == filepath.Clean(keyPath)) {
			return true
		}
	}
	return false
}

func (n *nodeRuntime) manageNodeCertificate(action string, request nodeCertificateManageRequest) (json.RawMessage, error) {
	if !fleet.SafeName(request.Name) {
		return nil, errors.New("invalid certificate name")
	}
	n.certificateMu.Lock()
	defer n.certificateMu.Unlock()
	certificates, err := readNodeManagedCertificates(n.manager.dataDir)
	if err != nil {
		return nil, err
	}
	metadata, ok := certificates[request.Name]
	if !ok {
		if action == "delete" {
			return json.Marshal(map[string]bool{"deleted": true})
		}
		return nil, errors.New("unknown node-local certificate")
	}
	if action == "update" {
		if _, err = nodeCertificateCredentials(n.manager.dataDir, metadata.DNS, request.CredentialName); err != nil {
			return nil, err
		}
		if metadata.CredentialName != request.CredentialName {
			if err = clearFleetCloudflareCache(filepath.Join(n.manager.dataDir, "certificates", request.Name, "acme")); err != nil {
				return nil, err
			}
		}
		metadata.Automatic = request.Automatic
		metadata.CredentialName = request.CredentialName
		certificates[request.Name] = metadata
		if err = saveNodeManagedCertificates(n.manager.dataDir, certificates); err != nil {
			return nil, err
		}
		return json.Marshal(metadata)
	}
	if action != "delete" {
		return nil, errors.New("unknown certificate operation")
	}
	n.mu.Lock()
	n.manager.mu.Lock()
	used := stateUsesNodeCertificate(n.manager.state, n.manager.dataDir, request.Name)
	n.manager.mu.Unlock()
	if n.applied != nil {
		var applied State
		if err = json.Unmarshal(n.applied.Model, &applied); err != nil {
			n.mu.Unlock()
			return nil, err
		}
		used = used || stateUsesNodeCertificate(applied, n.manager.dataDir, request.Name)
	}
	if used {
		n.mu.Unlock()
		return nil, errors.New("certificate is used by a TLS inbound; replace or remove its binding first")
	}
	defer n.mu.Unlock()
	// Move files out of service before changing metadata; restore on write failure.
	dir := filepath.Join(n.manager.dataDir, "certificates", request.Name)
	trash := dir + ".deleted-" + fleet.ID()
	moved := false
	if err = os.Rename(dir, trash); err == nil {
		moved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	delete(certificates, request.Name)
	if err = saveNodeManagedCertificates(n.manager.dataDir, certificates); err != nil {
		if moved {
			_ = os.Rename(trash, dir)
		}
		return nil, err
	}
	if moved {
		if err = os.RemoveAll(trash); err != nil {
			return nil, err
		}
	}
	return json.Marshal(map[string]bool{"deleted": true})
}

func (a *App) handleFleetNodeCertificateManage(w http.ResponseWriter, r *http.Request, nodeID, action string) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, apiResponse{})
		return
	}
	var request nodeCertificateManageRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, 400, apiResponse{Message: err.Error()})
		return
	}
	if !fleet.SafeName(request.Name) {
		writeJSON(w, 400, apiResponse{Message: "invalid certificate name"})
		return
	}
	f := a.fleet
	f.mu.Lock()
	defer f.mu.Unlock()
	node := f.store.Node(nodeID)
	if node == nil || node.Revoked {
		writeJSON(w, 404, apiResponse{Message: "unknown or revoked child server"})
		return
	}
	if action == "delete" {
		editor, err := f.editor(nodeID)
		if err != nil {
			writeJSON(w, 400, apiResponse{Message: err.Error()})
			return
		}
		editor.manager.mu.Lock()
		used := stateUsesNodeCertificate(editor.manager.state, "", request.Name)
		editor.manager.mu.Unlock()
		if node.Desired != nil {
			var desired State
			if err = json.Unmarshal(node.Desired.Model, &desired); err != nil {
				writeJSON(w, 400, apiResponse{Message: err.Error()})
				return
			}
			used = used || stateUsesNodeCertificate(desired, "", request.Name)
		}
		if used {
			writeJSON(w, 409, apiResponse{Message: "certificate is used by a TLS inbound; replace or remove its binding first"})
			return
		}
	}
	payload, _ := json.Marshal(request)
	task, err := f.store.Enqueue(nodeID, "cert-"+action, fleet.ID(), payload, 20*time.Minute)
	if err != nil {
		writeJSON(w, 409, apiResponse{Message: err.Error()})
		return
	}
	writeJSON(w, 202, apiResponse{OK: true, Data: map[string]any{"task": task}})
}
