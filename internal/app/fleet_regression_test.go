package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFleetPolicyDigestIgnoresRuntimeTimestamps(t *testing.T) {
	a := fleetTestApp(t)
	f := a.fleet
	node, err := f.store.AddNode("one", "one.example")
	if err != nil {
		t.Fatal(err)
	}
	ed, err := f.editor(node.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = ed.manager.saveInbound(Inbound{ID: "in", Name: "socks", Type: "socks", Listen: "0.0.0.0", Port: 18080, Enabled: true, Clients: []Client{{ID: "user", Name: "alice", Username: "alice", Password: "original", Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	first, err := f.publish(node.ID, ed, nil)
	if err != nil {
		t.Fatal(err)
	}
	ed.manager.mu.Lock()
	ed.manager.state.UpdatedAt = "different"
	ed.manager.state.Inbounds[0].Traffic.Down = 999
	ed.manager.state.Inbounds[0].Clients[0].LastOnline = "different"
	ed.manager.mu.Unlock()
	second, err := f.publish(node.ID, ed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || first.Revision != second.Revision {
		t.Fatal("runtime-only changes altered policy revision")
	}
}

func testFleetCertificate(t *testing.T) fleetCertificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "naive.example"}, DNSNames: []string{"naive.example"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(48 * time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, leaf, leaf, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return fleetCertificate{Name: "test", Domain: "naive.example", Certificate: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}))}
}

func TestFleetCentralCertificateSidecarBindingAndSecrecy(t *testing.T) {
	a := fleetTestApp(t)
	cert := testFleetCertificate(t)
	data, _ := json.Marshal(cert)
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/fleet/certificates/import", data))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	node, err := a.fleet.store.AddNode("one", "naive.example")
	if err != nil {
		t.Fatal(err)
	}
	desired, err := a.fleet.createSidecar(node.ID, fleetSidecarInput{Kind: "naive", Port: 443, Username: "alice", Password: "secret", Domain: "naive.example", CertificateName: cert.Name})
	if err != nil {
		t.Fatal(err)
	}
	if len(desired.Sidecars) != 1 {
		t.Fatal("Naive sidecar missing")
	}
	var config struct {
		Apps struct {
			HTTP struct {
				Servers map[string]struct {
					Routes []struct {
						Handle []struct {
							Credentials [][]byte `json:"auth_credentials"`
						} `json:"handle"`
					} `json:"routes"`
				} `json:"servers"`
			} `json:"http"`
		} `json:"apps"`
	}
	if err = json.Unmarshal([]byte(desired.Sidecars[0].Config), &config); err != nil {
		t.Fatal(err)
	}
	credentials := config.Apps.HTTP.Servers["naive"].Routes[0].Handle[0].Credentials
	if len(credentials) != 1 || string(credentials[0]) != base64.StdEncoding.EncodeToString([]byte("alice:secret")) {
		t.Fatal("credential encoding differs from the upstream forwardproxy byte-array contract")
	}
	registry, err := a.fleet.certificates()
	if err != nil || len(registry[cert.Name].Bindings) != 1 {
		t.Fatal("Naive certificate renewal binding missing", err)
	}
	if err = a.fleet.distributeCertificate(registry[cert.Name]); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest("GET", "/api/fleet/certificates", nil))
	if w.Code != 200 || strings.Contains(w.Body.String(), "BEGIN PRIVATE KEY") {
		t.Fatal("certificate list exposed private material")
	}
	_, err = a.fleet.createSidecar(node.ID, fleetSidecarInput{Kind: "shadowtls", Port: 8443, Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	node = a.fleet.store.Node(node.ID)
	if len(node.Desired.Sidecars) != 3 || node.Desired.Sidecars[1].Kind != "shadowsocks" {
		t.Fatal("managed backend missing")
	}
}

func TestFleetMeritFullStoreRepeatImportAndDuplicateRejection(t *testing.T) {
	a := fleetTestApp(t)
	input := map[string]any{"version": 1, "settings": map[string]string{"adminUsername": "ignored"}, "tasks": []any{}, "nodes": []any{map[string]any{"id": "legacy", "name": "original", "apiKey": "old-secret", "address": "one.example", "ports": []any{map[string]any{"id": "p", "port": 12345, "protocol": "TCP", "proxyType": "mieru", "username": "alice", "password": "original", "enabled": true, "trafficUsed": 1234}}}}}
	data, _ := json.Marshal(input)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		a.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/fleet/import/merit", data))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	nodes := a.fleet.store.Nodes()
	if len(nodes) != 1 || nodes[0].Desired == nil {
		t.Fatal("repeat import duplicated or lost node")
	}
	if strings.Contains(string(nodes[0].Desired.Model), "old-secret") {
		t.Fatal("legacy API key migrated")
	}
	input["nodes"] = append(input["nodes"].([]any), input["nodes"].([]any)[0])
	data, _ = json.Marshal(input)
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, fleetAuthRequest("POST", "/api/fleet/import/merit", data))
	if w.Code != 400 {
		t.Fatal("duplicate legacy nodes accepted")
	}
}
