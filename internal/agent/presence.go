package agent

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

var publicIPProviders = []string{
	"https://api.ipify.org", "https://ifconfig.me/ip", "https://ip.sb", "https://api64.ipify.org",
}

func discoverPublicIP(ctx context.Context) string {
	client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return probePublicIP(ctx, client, publicIPProviders)
}

func probePublicIP(ctx context.Context, client *http.Client, providers []string) string {
	for _, provider := range providers {
		if ctx.Err() != nil {
			break
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider, nil)
		if err != nil {
			continue
		}
		response, err := client.Do(req)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, 129))
		response.Body.Close()
		if err != nil || response.StatusCode != http.StatusOK || len(data) > 128 {
			continue
		}
		if ip := fleet.PublicIP(string(data)); ip != "" {
			return ip
		}
	}
	return ""
}

func (a *Agent) address() string {
	a.addressMu.RLock()
	defer a.addressMu.RUnlock()
	return a.publicIP
}

func (a *Agent) snapshot() fleet.Status {
	status := a.Runtime.Snapshot()
	status.EncryptionPublicKey = a.Client.Config.Encryption.Public()
	if ip := a.address(); ip != "" {
		status.PublicIP = ip
	}
	return status
}

func (a *Agent) discoverAddresses(ctx context.Context, discover func(context.Context) string) {
	for ctx.Err() == nil {
		if ip := fleet.PublicIP(discover(ctx)); ip != "" {
			a.addressMu.Lock()
			a.publicIP = ip
			a.addressMu.Unlock()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
		}
	}
}

func (a *Agent) heartbeatLoop(ctx context.Context, interval time.Duration) {
	// A separate client and replay guard avoid racing with poll/config refresh.
	client := &Client{HTTP: a.Client.HTTP}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for ctx.Err() == nil {
		cfg, err := LoadConfig(a.Dir)
		if err != nil {
			return
		}
		if _, err := os.Stat(filepath.Join(a.Dir, "break-glass")); err == nil {
			return
		}
		client.Config = cfg
		reportCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		var out map[string]bool
		err = client.request(reportCtx, "/fleet/v1/heartbeat", cfg.NodeID, fleet.Heartbeat{PublicIP: a.address()}, &out)
		cancel()
		if errors.Is(err, ErrAccessRevoked) {
			a.wipeRevokedAccess()
			return
		}
		if err != nil && ctx.Err() == nil {
			log.Printf("fleet heartbeat: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
