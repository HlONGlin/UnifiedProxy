package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/agent"
)

// Opt-in loopback transport test using actual Mihomo on both ends. No external
// proxy, DNS, CA or provider is needed. The oldest backup shape is the input.
func TestLegacyMihomoTransportFromOldestBackup(t *testing.T) {
	core := os.Getenv("UNIFIEDPROXY_TEST_REAL_CORE")
	if core == "" {
		t.Skip("set UNIFIEDPROXY_TEST_REAL_CORE to an actual Linux Mihomo binary")
	}
	freePort := func() int {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		return l.Addr().(*net.TCPAddr).Port
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "legacy-credentials-reached-origin") }))
	defer target.Close()
	for _, protocol := range []string{"oldest-mieru-tcp", "mieru-udp", "anytls", "shadowtls"} {
		t.Run(protocol, func(t *testing.T) {
			m := &CoreManager{dataDir: t.TempDir(), connectionStats: map[string]connectionSample{}, enforcedClients: map[string]bool{}}
			if err := m.load(); err != nil {
				t.Fatal(err)
			}
			m.state.Settings.CorePath = core
			m.state.Settings.APIAddress = fmt.Sprintf("127.0.0.1:%d", freePort())
			m.state.Settings.PanelPath = "/"
			defer m.stopCore()
			runtime := &nodeRuntime{manager: m, sidecars: &agent.Sidecars{Dir: m.dataDir}}
			defer runtime.sidecars.Stop()
			a := &App{manager: m, standalone: &standaloneRuntime{runtime: runtime}, session: "fleet-test-session"}
			var old map[string]any
			_ = json.Unmarshal([]byte(oldestLegacyBackup), &old)
			listener := old["nodes"].([]any)[0].(map[string]any)["ports"].([]any)[0].(map[string]any)
			listener["port"] = freePort()
			if protocol == "mieru-udp" {
				listener["protocol"] = "UDP"
			}
			if protocol == "anytls" {
				cert := testFleetCertificate(t)
				listener["proxyType"] = "anytls"
				listener["domain"] = cert.Domain
				listener["tlsCert"] = cert.Certificate
				listener["tlsKey"] = cert.PrivateKey
			}
			if protocol == "shadowtls" {
				handshake := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "local-handshake") }))
				handshake.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
				handshake.StartTLS()
				defer handshake.Close()
				listener["proxyType"] = "shadowtls"
				listener["domain"] = "localhost"
				listener["handshakeAddr"] = handshake.Listener.Addr().String()
				listener["backendManaged"] = true
				listener["backendAddr"] = fmt.Sprintf("127.0.0.1:%d", freePort())
				listener["backendPassword"] = "original-backend-password"
			}
			raw, _ := json.Marshal(old)
			response := importLegacyTest(t, a, raw)
			if response.Code != 200 {
				t.Fatal(response.Code, response.Body.String())
			}
			var proxy map[string]any
			if protocol == "shadowtls" {
				for _, sidecar := range runtime.applied.Sidecars {
					if sidecar.Kind != "shadowtls" {
						continue
					}
					if sidecar.Password != "original-password" || sidecar.BackendPassword != "original-backend-password" || sidecar.Cipher != "aes-256-gcm" {
						t.Fatal("original ShadowTLS credentials/cipher changed")
					}
					proxy = fleetSidecarProxies(subscriptionSnapshot{FleetSidecars: []fleetSubscriptionSidecar{{NodeName: "loopback", Server: "127.0.0.1", Config: sidecar}}}, "127.0.0.1")[0]
					proxy["plugin-opts"].(map[string]any)["skip-cert-verify"] = true // Self-signed loopback handshake only.
				}
			} else {
				inbound := m.state.Inbounds[0]
				client := inbound.Clients[0]
				if client.Password != "original-password" || client.Username != "old-alice" {
					t.Fatal("original credentials changed")
				}
				proxy = subscriptionProxy(inbound, client, "127.0.0.1")
			}
			if proxy == nil {
				t.Fatal("no migrated client proxy")
			}
			if protocol == "anytls" {
				proxy["skip-cert-verify"] = true
			} // Generated self-signed fixture only.
			clientPort := freePort()
			config := map[string]any{"mixed-port": clientPort, "bind-address": "127.0.0.1", "mode": "rule", "log-level": "info", "proxies": []any{proxy}, "rules": []string{"MATCH," + proxy["name"].(string)}}
			dir := t.TempDir()
			path := filepath.Join(dir, "client.json")
			encoded, _ := json.Marshal(config)
			if err := os.WriteFile(path, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			log, err := os.Create(filepath.Join(dir, "client.log"))
			if err != nil {
				t.Fatal(err)
			}
			defer log.Close()
			cmd := exec.Command(core, "-d", dir, "-f", path)
			cmd.Stdout = log
			cmd.Stderr = log
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
			proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", clientPort))
			transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
			defer transport.CloseIdleConnections()
			httpClient := &http.Client{Transport: transport, Timeout: 5 * time.Second}
			deadline := time.Now().Add(15 * time.Second)
			var lastErr error
			for time.Now().Before(deadline) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				req, _ := http.NewRequestWithContext(ctx, "GET", target.URL, nil)
				result, requestErr := httpClient.Do(req)
				if requestErr == nil {
					body, _ := io.ReadAll(result.Body)
					result.Body.Close()
					cancel()
					if string(body) == "legacy-credentials-reached-origin" {
						t.Log("actual Mihomo client → migrated listener → loopback HTTP origin passed")
						return
					}
					lastErr = fmt.Errorf("unexpected response %s", body)
				} else {
					lastErr = requestErr
					cancel()
				}
				time.Sleep(100 * time.Millisecond)
			}
			output, _ := os.ReadFile(filepath.Join(dir, "client.log"))
			t.Fatal(lastErr, strings.TrimSpace(string(output)))
		})
	}
}
