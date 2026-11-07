package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFleetCertificateIssueBootstrapsACMEAndKeepsTokenOutOfRegistry(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("fake ACME executable uses a POSIX shell")
	}
	a := fleetTestApp(t)
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("UNIFIEDPROXY_ACME_PATH", "")
	t.Setenv("ACME_TEST_INSTALL_ARGS", filepath.Join(tmp, "install-args"))
	t.Setenv("ACME_TEST_ARGS", filepath.Join(tmp, "acme-args"))
	t.Setenv("ACME_TEST_TOKEN", filepath.Join(tmp, "acme-token"))
	t.Setenv("ACME_TEST_RENEW_TOKEN", filepath.Join(tmp, "renew-token"))
	t.Setenv("CF_Token", "")
	fixture := testFleetCertificate(t)
	fixtureCert := filepath.Join(tmp, "fixture.crt")
	fixtureKey := filepath.Join(tmp, "fixture.key")
	if err := os.WriteFile(fixtureCert, []byte(fixture.Certificate), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixtureKey, []byte(fixture.PrivateKey), 0600); err != nil {
		t.Fatal(err)
	}
	node, err := a.fleet.store.AddNode("anytls", "node.example")
	if err != nil {
		t.Fatal(err)
	}
	editor, err := a.fleet.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = editor.manager.saveInbound(Inbound{ID: "anytls-in", Name: "AnyTLS", Type: "anytls", Listen: "0.0.0.0", Port: 8443, Enabled: true, TLS: true, TLSServerName: fixture.Domain, Certificate: fixture.Certificate, PrivateKey: fixture.PrivateKey, CertificateMode: "content", Clients: []Client{{ID: "user", Name: "alice", Password: "secret", Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ACME_TEST_CERT", fixtureCert)
	t.Setenv("ACME_TEST_KEY", fixtureKey)
	installer := `#!/bin/sh
printf '%s\n' "$*" > "$ACME_TEST_INSTALL_ARGS"
home=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --home) home="$2"; shift 2 ;;
    *) shift ;;
  esac
done
mkdir -p "$home"
cat > "$home/acme.sh" <<'ACMESCRIPT'
#!/bin/sh
printf '%s\n' "$*" >> "$ACME_TEST_ARGS"
if [ "$1" = "--issue" ]; then printf '%s' "$CF_Token" > "$ACME_TEST_TOKEN"; fi
if [ "$1" = "--renew" ]; then printf '%s' "$CF_Token" > "$ACME_TEST_RENEW_TOKEN"; fi
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
ACMESCRIPT
chmod 0700 "$home/acme.sh"
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(installer))
	}))
	defer server.Close()
	previousURL := fleetACMEScriptURL
	previousChecksum := fleetACMEScriptSHA256
	digest := sha256.Sum256([]byte(installer))
	fleetACMEScriptURL = server.URL
	fleetACMEScriptSHA256 = hex.EncodeToString(digest[:])
	t.Cleanup(func() {
		fleetACMEScriptURL = previousURL
		fleetACMEScriptSHA256 = previousChecksum
	})
	token := "test-cloudflare-token-do-not-store"
	body, err := json.Marshal(map[string]any{
		"name": "issued", "domain": fixture.Domain, "dns": "dns_cf", "automatic": true,
		"dnsCredentials": map[string]string{"cloudflareToken": token},
		"bindings":       []map[string]string{{"nodeId": node.ID, "inboundId": "anytls-in"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest(http.MethodPost, "/api/fleet/certificates/issue", body))
	if w.Code != http.StatusOK {
		t.Fatalf("issue status %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), token) {
		t.Fatal("issuance response exposed the Cloudflare token")
	}
	node = a.fleet.store.Node(node.ID)
	if node == nil || node.Desired == nil || len(node.Desired.Certificates) != 1 {
		t.Fatal("one-click issuance did not publish the certificate to the bound AnyTLS node")
	}
	args, err := os.ReadFile(os.Getenv("ACME_TEST_ARGS"))
	if err != nil {
		t.Fatal(err)
	}
	configHome := filepath.Join(a.manager.dataDir, "acme", "config")
	if !strings.Contains(string(args), "--dns dns_cf") || !strings.Contains(string(args), "--config-home "+configHome) {
		t.Fatalf("ACME issue did not use DNS challenge and persistent config home: %s", args)
	}
	installArgs, err := os.ReadFile(os.Getenv("ACME_TEST_INSTALL_ARGS"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installArgs), "--nocron") || !strings.Contains(string(installArgs), "--config-home "+configHome) {
		t.Fatalf("ACME bootstrap options were incomplete: %s", installArgs)
	}
	usedToken, err := os.ReadFile(os.Getenv("ACME_TEST_TOKEN"))
	if err != nil || string(usedToken) != token {
		t.Fatalf("Cloudflare token was not passed to the issue command: %q, %v", usedToken, err)
	}
	registry, err := os.ReadFile(filepath.Join(a.manager.dataDir, fleetCertificateFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(registry), token) {
		t.Fatal("Cloudflare token was persisted in the certificate registry")
	}
	certificates, err := a.fleet.certificates()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.fleet.acquireCertificate(context.Background(), certificates["issued"], true, fleetDNSCredentials{}); err != nil {
		t.Fatal(err)
	}
	renewArgs, err := os.ReadFile(os.Getenv("ACME_TEST_ARGS"))
	if err != nil || !strings.Contains(string(renewArgs), "--renew") || !strings.Contains(string(renewArgs), "--config-home "+configHome) {
		t.Fatalf("renewal did not use persistent ACME state: %s, %v", renewArgs, err)
	}
	renewToken, err := os.ReadFile(os.Getenv("ACME_TEST_RENEW_TOKEN"))
	if err != nil || len(renewToken) != 0 {
		t.Fatalf("Cloudflare token was passed directly to renewal: %q, %v", renewToken, err)
	}
	info, err := os.Stat(configHome)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("ACME configuration directory permissions are not private: %v, %v", info, err)
	}
	w = httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest(http.MethodGet, "/api/fleet/certificates", nil))
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), token) {
		t.Fatal("certificate list exposed Cloudflare credentials", w.Code, w.Body.String())
	}
}

func TestFleetCertificateBindingPublishesAnyTLSToChildServer(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("anytls", "node.example")
	if err != nil {
		t.Fatal(err)
	}
	editor, err := a.fleet.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	certificate := testFleetCertificate(t)
	inbound := Inbound{ID: "anytls-in", Name: "AnyTLS", Type: "anytls", Listen: "0.0.0.0", Port: 8443, Enabled: true, TLS: true, TLSServerName: "naive.example", Certificate: certificate.Certificate, PrivateKey: certificate.PrivateKey, CertificateMode: "content", Clients: []Client{{ID: "user", Name: "alice", Password: "secret", Enabled: true}}}
	if err = editor.manager.saveInbound(inbound); err != nil {
		t.Fatal(err)
	}
	certificate.Bindings = []fleetCertificateBinding{{NodeID: node.ID, InboundID: inbound.ID}}
	if err = a.fleet.distributeCertificate(certificate); err != nil {
		t.Fatal(err)
	}
	node = a.fleet.store.Node(node.ID)
	if node == nil || node.Desired == nil || len(node.Desired.Certificates) != 1 {
		t.Fatal("bound AnyTLS certificate was not published to the child node")
	}
	if !strings.Contains(node.Desired.Certificates[0].Certificate, "BEGIN CERTIFICATE") || !strings.Contains(node.Desired.Certificates[0].PrivateKey, "PRIVATE KEY") {
		t.Fatal("published child certificate bundle is incomplete")
	}
	if !strings.Contains(string(node.Desired.Model), "__FLEET_CERT_DIR__/inbound-anytls-in.crt") {
		t.Fatalf("child AnyTLS model does not reference the distributed certificate: %s", node.Desired.Model)
	}
}

func TestFleetCertificateBindingRejectsNonTLSAndMismatchedSNI(t *testing.T) {
	a := fleetTestApp(t)
	node, err := a.fleet.store.AddNode("node", "node.example")
	if err != nil {
		t.Fatal(err)
	}
	editor, err := a.fleet.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	certificate := testFleetCertificate(t)
	cases := []struct {
		name    string
		inbound Inbound
	}{
		{name: "non TLS", inbound: Inbound{ID: "plain", Name: "plain", Type: "socks", Port: 1080, Enabled: true}},
		{name: "wrong SNI", inbound: Inbound{ID: "wrong", Name: "AnyTLS", Type: "anytls", Port: 8443, Enabled: true, TLS: true, TLSServerName: "other.example", Certificate: certificate.Certificate, PrivateKey: certificate.PrivateKey, CertificateMode: "content"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := editor.manager.saveInbound(test.inbound); err != nil {
				t.Fatal(err)
			}
			bound := certificate
			bound.Bindings = []fleetCertificateBinding{{NodeID: node.ID, InboundID: test.inbound.ID}}
			if err := a.fleet.distributeCertificate(bound); err == nil {
				t.Fatal("invalid TLS certificate binding was accepted")
			}
		})
	}
}

func TestFleetACMEInstallerRejectsChecksumMismatch(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "executed")
	t.Setenv("ACME_CHECKSUM_MARKER", marker)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#!/bin/sh\n: > \"$ACME_CHECKSUM_MARKER\"\n"))
	}))
	defer server.Close()
	previousURL := fleetACMEScriptURL
	previousChecksum := fleetACMEScriptSHA256
	fleetACMEScriptURL = server.URL
	fleetACMEScriptSHA256 = strings.Repeat("0", 64)
	t.Cleanup(func() {
		fleetACMEScriptURL = previousURL
		fleetACMEScriptSHA256 = previousChecksum
	})
	dataDir := t.TempDir()
	err := installFleetACME(context.Background(), filepath.Join(dataDir, "home"), filepath.Join(dataDir, "config"))
	if err == nil || !strings.Contains(err.Error(), "SHA-256 verification") {
		t.Fatalf("installer accepted an unpinned script: %v", err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("unverified installer executed: %v", err)
	}
}
