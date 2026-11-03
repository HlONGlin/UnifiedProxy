package app

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/RomanovCaesar/m-ui/internal/agent"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// Opt in because this downloads pinned upstream releases and starts real
// processes. All listeners and forwarded targets remain on local loopback.
func TestFleetRealSidecarProcesses(t *testing.T) {
	if runtime.GOOS != "linux" || os.Getenv("MUI_FLEET_INTEGRATION") != "1" {
		t.Skip("set MUI_FLEET_INTEGRATION=1 on Linux for upstream sidecar integration")
	}
	freePort := func() int {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		_ = l.Close()
		return port
	}
	naivePort, backendPort, shadowPort := freePort(), freePort(), freePort()
	cert := testFleetCertificate(t)
	_, configs, _, err := convertMerit(meritNode{ID: "fixture", Ports: []meritPort{
		{ID: "naive", Port: naivePort, ProxyType: "naive", Domain: "naive.example", Username: "fixture", Password: "fixture-password", Enabled: true, TLSCert: cert.Certificate, TLSKey: cert.PrivateKey},
		{ID: "shadow", Port: shadowPort, ProxyType: "shadowtls", Username: "fixture", Password: "tls-password", Enabled: true, BackendManaged: true, BackendAddr: net.JoinHostPort("127.0.0.1", strconv.Itoa(backendPort)), BackendPassword: "backend-password", BackendCipher: "aes-128-gcm"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// Permit only the test's loopback target; production keeps the module ACL.
	var cfg map[string]any
	_ = json.Unmarshal([]byte(configs[0].Config), &cfg)
	server := cfg["apps"].(map[string]any)["http"].(map[string]any)["servers"].(map[string]any)["naive"].(map[string]any)
	server["listen"] = []string{net.JoinHostPort("127.0.0.1", strconv.Itoa(naivePort))}
	server["automatic_https"] = map[string]bool{"disable": true}
	route := server["routes"].([]any)[0].(map[string]any)
	handles := route["handle"].([]any)
	handles[0].(map[string]any)["acl"] = []any{map[string]any{"subjects": []string{"127.0.0.1"}, "allow": true}}
	route["handle"] = append(handles, map[string]any{"handler": "static_response", "status_code": 404})
	data, _ := json.Marshal(cfg)
	configs[0].Config = string(data)
	sidecars := &agent.Sidecars{Dir: t.TempDir()}
	defer sidecars.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	if err = sidecars.Validate(ctx, configs); err != nil {
		t.Fatal(err)
	}
	if err = sidecars.Apply(configs); err != nil {
		t.Fatal(err)
	}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "fleet-sidecar-ok") }))
	defer target.Close()
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM([]byte(cert.Certificate))
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(naivePort))
	token := base64.StdEncoding.EncodeToString([]byte("fixture:fixture-password"))
	for _, auth := range []string{"wrong", token} {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", address, &tls.Config{RootCAs: roots, ServerName: "naive.example", NextProtos: []string{"http/1.1"}})
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		crlf := string([]byte{13, 10})
		targetAddress := target.Listener.Addr().String()
		_, err = fmt.Fprintf(conn, "CONNECT %s HTTP/1.1%sHost: %s%sProxy-Authorization: Basic %s%s%s", targetAddress, crlf, targetAddress, crlf, auth, crlf, crlf)
		if err != nil {
			t.Fatal(err)
		}
		reader := bufio.NewReader(conn)
		res, err := http.ReadResponse(reader, &http.Request{Method: "CONNECT"})
		if err != nil {
			t.Fatal(err)
		}
		if auth == token {
			if res.StatusCode != 200 {
				t.Fatalf("authenticated CONNECT HTTP %d", res.StatusCode)
			}
			_, _ = fmt.Fprintf(conn, "GET / HTTP/1.1%sHost: %s%sConnection: close%s%s", crlf, targetAddress, crlf, crlf, crlf)
			forward, err := http.ReadResponse(reader, &http.Request{Method: "GET"})
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(forward.Body)
			_ = forward.Body.Close()
			if string(body) != "fleet-sidecar-ok" {
				t.Fatal("tunnel did not forward data")
			}
		} else if res.StatusCode == 200 {
			t.Fatal("invalid credentials accepted")
		}
		_ = conn.Close()
	}
	for _, port := range []int{backendPort, shadowPort} {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
		if err != nil {
			t.Fatal("backend or ShadowTLS listener failed", err)
		}
		_ = conn.Close()
	}
	sidecars.Stop()
	if err = sidecars.Apply(configs); err != nil {
		t.Fatal("sidecar restart failed", err)
	}
}
