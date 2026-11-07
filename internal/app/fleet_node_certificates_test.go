package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestResolveNodeCertificateReferenceLoadsOnlyLocalFiles(t *testing.T) {
	certificate := testFleetCertificate(t)
	dataDir := t.TempDir()
	certPath, keyPath, err := nodeCertificatePaths(dataDir, "edge-cert")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(certPath, []byte(certificate.Certificate), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(keyPath, []byte(certificate.PrivateKey), 0600); err != nil {
		t.Fatal(err)
	}
	inbound := Inbound{TLS: true, TLSServerName: certificate.Domain, Certificate: nodeCertificateReference("edge-cert", ".crt"), PrivateKey: nodeCertificateReference("edge-cert", ".key")}
	if err = resolveNodeCertificateReference(dataDir, &inbound); err != nil {
		t.Fatal(err)
	}
	if inbound.Certificate != filepath.ToSlash(certPath) || inbound.PrivateKey != filepath.ToSlash(keyPath) || inbound.CertificateMode != "file" {
		t.Fatalf("local certificate references resolved incorrectly: %#v", inbound)
	}
	if strings.Contains(inbound.Certificate, "BEGIN CERTIFICATE") || strings.Contains(inbound.PrivateKey, "PRIVATE KEY") {
		t.Fatal("node model contains certificate material instead of local paths")
	}
	inbound.TLSServerName = "wrong.example"
	inbound.Certificate = nodeCertificateReference("edge-cert", ".crt")
	inbound.PrivateKey = nodeCertificateReference("edge-cert", ".key")
	if err = resolveNodeCertificateReference(dataDir, &inbound); err == nil {
		t.Fatal("node accepted a local certificate that does not cover inbound SNI")
	}
}

func TestNodeCertificateIssueAndRenewStayOnChild(t *testing.T) {
	certificate := testFleetCertificate(t)
	dataDir := t.TempDir()
	fixtureCert := filepath.Join(dataDir, "fixture.crt")
	fixtureKey := filepath.Join(dataDir, "fixture.key")
	if err := os.WriteFile(fixtureCert, []byte(certificate.Certificate), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixtureKey, []byte(certificate.PrivateKey), 0600); err != nil {
		t.Fatal(err)
	}
	argsPath := filepath.Join(dataDir, "acme-args")
	tokenPath := filepath.Join(dataDir, "acme-token")
	renewTokenPath := filepath.Join(dataDir, "acme-renew-token")
	token := "child-local-cloudflare-token"
	accountID := "child-cloudflare-account-id"
	zoneID := "child-cloudflare-zone-id"
	t.Setenv("UNIFIEDPROXY_ACME_PATH", filepath.Join(dataDir, "acme.sh"))
	t.Setenv("CF_Token", "")
	t.Setenv("CF_Account_ID", "")
	t.Setenv("CF_Zone_ID", "")
	t.Setenv("ACME_TEST_TOKEN", tokenPath)
	t.Setenv("ACME_TEST_RENEW_TOKEN", renewTokenPath)
	t.Setenv("ACME_TEST_CERT", fixtureCert)
	t.Setenv("ACME_TEST_KEY", fixtureKey)
	t.Setenv("ACME_TEST_ARGS", argsPath)
	acme := `#!/bin/sh
printf '%s\n' "$*" >> "$ACME_TEST_ARGS"
if [ "$1" = "--issue" ]; then printf '%s|%s|%s' "$CF_Token" "$CF_Account_ID" "$CF_Zone_ID" > "$ACME_TEST_TOKEN"; exit 0; fi
if [ "$1" = "--renew" ]; then printf '%s|%s|%s' "$CF_Token" "$CF_Account_ID" "$CF_Zone_ID" > "$ACME_TEST_RENEW_TOKEN"; exit 0; fi
if [ "$1" = "--install-cert" ]; then
  cert=""
  key=""
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --key-file) key="$2"; shift 2 ;;
      --fullchain-file) cert="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  cp "$ACME_TEST_CERT" "$cert"
  cp "$ACME_TEST_KEY" "$key"
fi
`
	acmePath := filepath.Join(dataDir, "acme.sh")
	if err := os.WriteFile(acmePath, []byte(acme), 0700); err != nil {
		t.Fatal(err)
	}
	credentials := fleetDNSCredentials{CloudflareToken: token, CloudflareAccountID: accountID, CloudflareZoneID: zoneID}
	if err := saveNodeDNSCredentials(dataDir, credentials); err != nil {
		t.Fatal(err)
	}
	node := &nodeRuntime{manager: &CoreManager{dataDir: dataDir}}
	result, err := node.issueNodeCertificate(context.Background(), nodeCertificateIssueRequest{Name: "edge-cert", Domain: certificate.Domain, DNS: "dns_cf", Automatic: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(result), "PRIVATE KEY") || strings.Contains(string(result), certificate.PrivateKey) || strings.Contains(string(result), token) {
		t.Fatal("node returned its private key in the task result")
	}
	var metadata nodeManagedCertificate
	if err = json.Unmarshal(result, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Name != "edge-cert" || metadata.Domain != certificate.Domain || !metadata.Automatic || len(metadata.InboundIDs) != 0 {
		t.Fatalf("unexpected local certificate metadata: %#v", metadata)
	}
	certPath, keyPath, err := nodeCertificatePaths(dataDir, metadata.Name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(certPath); err != nil {
		t.Fatal(err)
	}
	keyInfo, err := os.Stat(keyPath)
	if err != nil || keyInfo.Mode().Perm() != 0600 {
		t.Fatalf("local private key permissions are not private: %v, %v", keyInfo, err)
	}
	usedToken, err := os.ReadFile(tokenPath)
	if err != nil || string(usedToken) != token+"|"+accountID+"|"+zoneID {
		t.Fatalf("saved Cloudflare credentials were not passed to ACME: %q, %v", usedToken, err)
	}
	credentialPath := nodeDNSCredentialsPath(dataDir)
	credentialInfo, err := os.Stat(credentialPath)
	if err != nil || credentialInfo.Mode().Perm() != 0600 {
		t.Fatalf("local Cloudflare credentials are not protected: %v, %v", credentialInfo, err)
	}
	if !node.cloudflareCredentialsConfigured() {
		t.Fatal("node did not report saved Cloudflare credentials")
	}
	registry, err := os.ReadFile(nodeManagedCertificatesPath(dataDir))
	if err != nil || strings.Contains(string(registry), token) {
		t.Fatalf("node registry persisted the Cloudflare token: %v", err)
	}
	configInfo, err := os.Stat(filepath.Join(dataDir, "acme", "config"))
	if err != nil || configInfo.Mode().Perm() != 0700 {
		t.Fatalf("local ACME credentials directory is not private: %v, %v", configInfo, err)
	}
	argsBeforeConflict, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	certBeforeConflict, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = node.issueNodeCertificate(context.Background(), nodeCertificateIssueRequest{Name: metadata.Name, Domain: "other.example", InboundID: "other-in"}); err == nil || !strings.Contains(err.Error(), "already used for another domain") {
		t.Fatalf("conflicting certificate name was not rejected before issuance: %v", err)
	}
	argsAfterConflict, err := os.ReadFile(argsPath)
	if err != nil || string(argsAfterConflict) != string(argsBeforeConflict) {
		t.Fatalf("conflicting certificate name invoked ACME: %v", err)
	}
	certAfterConflict, err := os.ReadFile(certPath)
	if err != nil || string(certAfterConflict) != string(certBeforeConflict) {
		t.Fatalf("conflicting certificate name overwrote the existing certificate: %v", err)
	}
	if _, err = node.renewNodeCertificate(context.Background(), metadata.Name); err != nil {
		t.Fatal(err)
	}
	renewedToken, err := os.ReadFile(renewTokenPath)
	if err != nil || string(renewedToken) != token+"|"+accountID+"|"+zoneID {
		t.Fatalf("renewal did not reuse node-local Cloudflare credentials: %q, %v", renewedToken, err)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil || !strings.Contains(string(args), "--renew") || !strings.Contains(string(args), "--force") || !strings.Contains(string(args), "--dns dns_cf") {
		t.Fatalf("manual renewal did not run on the child: %s, %v", args, err)
	}
}

func TestFleetNodeCertificateIssueQueuesLocalOnlyMaterial(t *testing.T) {
	a := fleetTestApp(t)
	certificate := testFleetCertificate(t)
	node, err := a.fleet.store.AddNode("edge", certificate.Domain)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.fleet.store.UpdateStatus(node.ID, fleet.Status{Version: appVersion, Protocol: fleet.ProtocolVersion, Capabilities: append([]string(nil), fleet.TaskKinds...)}); err != nil {
		t.Fatal(err)
	}
	editor, err := a.fleet.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	inbound := Inbound{ID: "anytls-in", Name: "AnyTLS", Type: "anytls", Listen: "0.0.0.0", Port: 8443, Enabled: true, TLS: true, TLSServerName: certificate.Domain, Certificate: certificate.Certificate, PrivateKey: certificate.PrivateKey, CertificateMode: "content"}
	if err = editor.manager.saveInbound(inbound); err != nil {
		t.Fatal(err)
	}
	if _, err = a.fleet.publish(node.ID, editor, nil); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(nodeCertificateIssueRequest{Name: "edge-cert", Domain: certificate.Domain, Automatic: true, InboundID: inbound.ID})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest(http.MethodPost, "/api/fleet/nodes/"+node.ID+"/certificates/issue", body))
	if w.Code != http.StatusAccepted {
		t.Fatalf("issue status %d: %s", w.Code, w.Body.String())
	}
	var response map[string]json.RawMessage
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var data map[string]json.RawMessage
	if err = json.Unmarshal(response["data"], &data); err != nil {
		t.Fatal(err)
	}
	var issueTask fleet.Task
	if err = json.Unmarshal(data["task"], &issueTask); err != nil {
		t.Fatal(err)
	}
	if issueTask.Kind != "cert-issue" || strings.Contains(string(issueTask.Payload), "PRIVATE KEY") || strings.Contains(string(issueTask.Payload), "cloudflareToken") {
		t.Fatalf("task payload contains controller-side certificate secrets: %#v", issueTask)
	}
	var queuedIssue nodeCertificateIssueRequest
	if err = json.Unmarshal(issueTask.Payload, &queuedIssue); err != nil || queuedIssue.TLSServerName != certificate.Domain {
		t.Fatalf("issue task did not carry the validated inbound SNI: %#v, %v", queuedIssue, err)
	}
	updatedNode := a.fleet.store.Node(node.ID)
	if updatedNode == nil || updatedNode.Desired == nil || len(updatedNode.Desired.Certificates) != 0 {
		t.Fatal("node-local certificate was included in controller desired state")
	}
	var desired State
	if err = json.Unmarshal(updatedNode.Desired.Model, &desired); err != nil {
		t.Fatal(err)
	}
	name, isReference, err := nodeCertificateReferenceName(desired.Inbounds[0])
	if err != nil || !isReference || name != "edge-cert" {
		t.Fatalf("desired inbound does not reference the local certificate: %#v, %v", desired.Inbounds[0], err)
	}
	var latestApply *fleet.Task
	for _, task := range a.fleet.store.Tasks() {
		if task.NodeID == node.ID && task.Kind == "apply-state" && task.Status == "pending" {
			var payload fleet.DesiredState
			if json.Unmarshal(task.Payload, &payload) == nil && payload.Digest == updatedNode.Desired.Digest {
				latestApply = task
			}
		}
	}
	if latestApply == nil || !issueTask.CreatedAt.Before(latestApply.CreatedAt) {
		t.Fatal("certificate issuance must run before the model references its local files")
	}
}

func TestFleetNodeCertificateCanBeIssuedWithoutTLSInbound(t *testing.T) {
	a := fleetTestApp(t)
	certificate := testFleetCertificate(t)
	node, err := a.fleet.store.AddNode("edge", certificate.Domain)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.fleet.store.UpdateStatus(node.ID, fleet.Status{Version: appVersion, Protocol: fleet.ProtocolVersion, Capabilities: []string{"cert-issue"}}); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(nodeCertificateIssueRequest{Name: "standalone-cert", Domain: certificate.Domain, Automatic: true})
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest(http.MethodPost, "/api/fleet/nodes/"+node.ID+"/certificates/issue", body))
	if w.Code != http.StatusAccepted {
		t.Fatalf("independent issue status %d: %s", w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			Task fleet.Task
		}
	}
	if err = json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var issued nodeCertificateIssueRequest
	if err = json.Unmarshal(response.Data.Task.Payload, &issued); err != nil || issued.Name != "standalone-cert" || issued.InboundID != "" || issued.TLSServerName != "" {
		t.Fatalf("independent certificate task unexpectedly references an inbound: %#v, %v", issued, err)
	}
	updated := a.fleet.store.Node(node.ID)
	if updated == nil || updated.Desired != nil {
		t.Fatal("issuing an unbound certificate unexpectedly published node configuration")
	}
}

func TestFleetNodeCertificateCanBindAfterIndependentIssue(t *testing.T) {
	a := fleetTestApp(t)
	certificate := testFleetCertificate(t)
	node, err := a.fleet.store.AddNode("edge", certificate.Domain)
	if err != nil {
		t.Fatal(err)
	}
	details, _ := json.Marshal(map[string]any{"managedCertificates": []nodeManagedCertificate{{Name: "edge-cert", Domain: certificate.Domain, ExpiresAt: time.Now().Add(90 * 24 * time.Hour)}}})
	if err = a.fleet.store.UpdateStatus(node.ID, fleet.Status{Version: appVersion, Protocol: fleet.ProtocolVersion, Capabilities: append([]string(nil), fleet.TaskKinds...), Details: details}); err != nil {
		t.Fatal(err)
	}
	editor, err := a.fleet.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	inbound := Inbound{ID: "vless-in", Name: "VLESS", Type: "vless", Listen: "0.0.0.0", Port: 8443, Enabled: false, TLS: true, TLSServerName: certificate.Domain, Clients: []Client{{ID: "client-1", Name: "alice", UUID: "00000000-0000-0000-0000-000000000001", Enabled: true}}}
	if err = editor.manager.saveInbound(inbound); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"name": "edge-cert", "inboundId": inbound.ID})
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest(http.MethodPost, "/api/fleet/nodes/"+node.ID+"/certificates/bind", body))
	if w.Code != http.StatusAccepted {
		t.Fatalf("bind status %d: %s", w.Code, w.Body.String())
	}
	updated := a.fleet.store.Node(node.ID)
	if updated == nil || updated.Desired == nil {
		t.Fatal("binding did not publish the child configuration")
	}
	var desired State
	if err = json.Unmarshal(updated.Desired.Model, &desired); err != nil {
		t.Fatal(err)
	}
	name, reference, err := nodeCertificateReferenceName(desired.Inbounds[0])
	if err != nil || !reference || name != "edge-cert" {
		t.Fatalf("inbound does not use the node-local certificate: %#v, %v", desired.Inbounds[0], err)
	}
	var bindTask, applyTask *fleet.Task
	for _, task := range a.fleet.store.Tasks() {
		if task.Kind == "cert-bind" {
			bindTask = task
		}
		if task.Kind == "apply-state" && task.Status == "pending" {
			applyTask = task
		}
	}
	if bindTask == nil || applyTask == nil || !bindTask.CreatedAt.Before(applyTask.CreatedAt) {
		t.Fatal("certificate metadata binding must be delivered before the referencing model")
	}
}

func TestFleetNodeCloudflareCredentialsAreEncryptedBeforeControllerStorage(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("edge", "edge.example")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := fleet.NewEncryptionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err = a.fleet.store.UpdateStatus(node.ID, fleet.Status{Version: appVersion, Protocol: fleet.ProtocolVersion, Capabilities: []string{"cert-credentials"}, EncryptionPublicKey: identity.Public()}); err != nil {
		t.Fatal(err)
	}
	taskID := fleet.ID()
	plaintext := []byte("{\"cloudflareToken\":\"browser-secret-token\",\"cloudflareAccountId\":\"browser-account\",\"cloudflareZoneId\":\"browser-zone\"}")
	ciphertext, err := fleet.EncryptTaskPayload(identity.Public(), node.ID, taskID, "cert-credentials", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"taskId": taskID, "payload": json.RawMessage(ciphertext)})
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest(http.MethodPost, "/api/fleet/nodes/"+node.ID+"/certificates/dns-credentials", body))
	if w.Code != http.StatusAccepted || strings.Contains(w.Body.String(), "browser-secret-token") {
		t.Fatalf("credential delivery response status %d or leaked plaintext: %s", w.Code, w.Body.String())
	}
	for label, data := range map[string][]byte{
		"fleet state": func() []byte { b, _ := os.ReadFile(filepath.Join(a.fleet.dataDir, "fleet.json")); return b }(),
		"backup":      func() []byte { b, _ := a.fleet.store.Backup(); return b }(),
	} {
		if bytes.Contains(data, plaintext) || bytes.Contains(data, []byte("browser-secret-token")) || bytes.Contains(data, []byte("cloudflareToken")) {
			t.Fatalf("controller %s contains plaintext Cloudflare credentials", label)
		}
	}
	tasks := a.fleet.store.Tasks()
	if len(tasks) != 1 || !tasks[0].Encrypted || len(tasks[0].Payload) != 0 {
		t.Fatal("task history exposed the encrypted Cloudflare credential payload")
	}
}
