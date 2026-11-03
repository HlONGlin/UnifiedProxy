package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/RomanovCaesar/m-ui/internal/fleet"
)

func testFleetEgress(ctx context.Context, proxyURL, target string) (json.RawMessage, error) {
	p, err := url.Parse(proxyURL)
	if err != nil || (p.Scheme != "socks5" && p.Scheme != "socks5h") {
		return nil, errors.New("SOCKS5 proxy URL required")
	}
	ip := net.ParseIP(p.Hostname())
	if p.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, errors.New("egress proxy must be node-local")
	}
	if target == "" {
		target = "https://api.ipify.org?format=json"
	}
	u, err := url.Parse(target)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, errors.New("HTTPS egress test URL required")
	}
	transport := &http.Transport{Proxy: http.ProxyURL(p)}
	defer transport.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return nil, err
	}
	response, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("egress endpoint HTTP %d", response.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil {
		return nil, err
	}
	if len(b) > 4096 {
		return nil, errors.New("egress response too large")
	}
	return json.Marshal(map[string]string{"response": string(b), "via": p.Host})
}

func stageFleetUpgrade(ctx context.Context, dir, source, checksum string) error {
	var existing fleetUpgrade
	if fleet.ReadJSON(filepath.Join(dir, "pending-upgrade.json"), &existing) == nil && existing.Status == "pending-restart" {
		return errors.New("an upgrade is already running")
	}
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return errors.New("upgrade requires HTTPS")
	}
	expected, err := hex.DecodeString(checksum)
	if err != nil || len(expected) != sha256.Size {
		return errors.New("valid SHA256 required")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", source, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" || len(via) > 10 {
			return errors.New("unsafe update redirect")
		}
		return nil
	}}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("upgrade HTTP %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, (128<<20)+1))
	if err != nil {
		return err
	}
	if len(b) > 128<<20 {
		return errors.New("upgrade too large")
	}
	h := sha256.Sum256(b)
	if !strings.EqualFold(hex.EncodeToString(h[:]), checksum) {
		return errors.New("upgrade checksum mismatch")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	if runtime.GOOS != "linux" {
		return errors.New("automatic binary replacement requires the Linux systemd installer")
	}
	// Keep replacement next to the executable for atomic same-volume renames.
	if err = fleet.WriteFile(executable+".next", b, 0755); err != nil {
		return err
	}
	check, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(check, executable+".next", "version")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("candidate cannot run on this platform: %w", err)
	}
	return fleet.WriteJSON(filepath.Join(dir, "pending-upgrade.json"), fleetUpgrade{Executable: executable, SHA256: strings.ToLower(checksum), Status: "staged"})
}
