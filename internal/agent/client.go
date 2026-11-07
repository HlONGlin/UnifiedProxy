// Package agent implements outbound-only fleet enrollment and task execution.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

const ConfigFile = "agent.json"

type Config struct {
	ControlURL string                   `json:"controlURL"`
	ControlKey string                   `json:"controlKey"`
	NodeID     string                   `json:"nodeId"`
	Identity   fleet.Identity           `json:"identity"`
	Encryption fleet.EncryptionIdentity `json:"encryption"`
}

func LoadConfig(dir string) (Config, error) {
	var c Config
	err := fleet.ReadJSON(filepath.Join(dir, ConfigFile), &c)
	if err != nil {
		return c, err
	}
	if err = c.Validate(); err != nil {
		return c, err
	}
	if c.Encryption.PrivateKey == "" {
		c.Encryption, err = fleet.NewEncryptionIdentity()
		if err != nil {
			return c, err
		}
		if err = fleet.WriteJSON(filepath.Join(dir, ConfigFile), c); err != nil {
			return c, fmt.Errorf("save node encryption identity: %w", err)
		}
	}
	return c, nil
}
func (c Config) Validate() error {
	if err := fleet.ValidateControlURL(c.ControlURL); err != nil {
		return err
	}
	if c.NodeID == "" || !fleet.SafeName(c.NodeID) || len(c.ControlKey) < 40 {
		return errors.New("invalid enrollment identity")
	}
	_, err := c.Identity.Key()
	if err != nil {
		return err
	}
	if c.Encryption.PrivateKey != "" {
		_, err = c.Encryption.Key()
	}
	return err
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return errors.New("fleet redirects are disabled; use agent relocate")
	}}
}

type Client struct {
	Config Config
	HTTP   *http.Client
	replay fleet.ReplayGuard
}

func (c *Client) request(ctx context.Context, path, sender string, payload, out any) error {
	e, err := fleet.Sign(c.Config.Identity, sender, "POST "+path, payload)
	if err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(c.Config.ControlURL, "/")+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = newHTTPClient()
	}
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("control returned HTTP %d", res.StatusCode)
	}
	b, err = io.ReadAll(io.LimitReader(res.Body, fleet.MaxMessage+1))
	if err != nil {
		return err
	}
	if len(b) > fleet.MaxMessage {
		return errors.New("oversized control response")
	}
	var response fleet.Envelope
	if err = json.Unmarshal(b, &response); err != nil {
		return err
	}
	if err = c.replay.Verify(response, c.Config.ControlKey, "control", "response:"+e.Nonce, time.Now()); err != nil {
		return err
	}
	var marker struct {
		Error string `json:"error"`
		Wipe  bool   `json:"wipe"`
	}
	if json.Unmarshal(response.Payload, &marker) == nil && marker.Error == "access-revoked" && marker.Wipe {
		return ErrAccessRevoked
	}
	return json.Unmarshal(response.Payload, out)
}

// ErrAccessRevoked means the controller signed a revocation telling this node to
// stop proxy processes and delete its data directory.
var ErrAccessRevoked = errors.New("fleet access revoked")

func Enroll(ctx context.Context, dir, url, key, token, name string) (Config, error) {
	if err := fleet.ValidateControlURL(url); err != nil {
		return Config{}, err
	}
	if token == "" || key == "" {
		return Config{}, errors.New("enrollment token and trusted control public key are required")
	}
	if _, err := os.Stat(filepath.Join(dir, ConfigFile)); err == nil {
		return Config{}, errors.New("node is already enrolled; leave before rebinding")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	identity, err := fleet.LoadIdentityWithLegacy(filepath.Join(dir, "agent-identity.json"), filepath.Join(dir, "multi-control.json"))
	if err != nil {
		return Config{}, err
	}
	encryption, err := fleet.NewEncryptionIdentity()
	if err != nil {
		return Config{}, err
	}
	cfg := Config{ControlURL: strings.TrimRight(url, "/"), ControlKey: key, Identity: identity, Encryption: encryption}
	client := &Client{Config: cfg}
	var response fleet.EnrollmentResponse
	if err = client.request(ctx, "/fleet/v1/enroll", fleet.Fingerprint(identity.Public()), fleet.Enrollment{Token: strings.TrimSpace(token), PublicKey: identity.Public(), EncryptionPublicKey: encryption.Public(), Name: name}, &response); err != nil {
		return Config{}, err
	}
	if response.PublicKey != key || (response.EncryptionPublicKey != "" && response.EncryptionPublicKey != encryption.Public()) || response.Protocol != fleet.ProtocolVersion || !fleet.SafeName(response.NodeID) {
		return Config{}, errors.New("enrollment response does not match trusted controller")
	}
	cfg.NodeID = response.NodeID
	if err = fleet.WriteJSON(filepath.Join(dir, ConfigFile), cfg); err != nil {
		return Config{}, fmt.Errorf("enrolled remotely but local identity save failed; issue a rebind token for node %s: %w", cfg.NodeID, err)
	}
	return cfg, nil
}

func Relocate(ctx context.Context, dir, url string) error {
	cfg, err := LoadConfig(dir)
	if err != nil {
		return err
	}
	if err = fleet.ValidateControlURL(url); err != nil {
		return err
	}
	cfg.ControlURL = strings.TrimRight(url, "/")
	client := &Client{Config: cfg}
	var response map[string]bool
	if err = client.request(ctx, "/fleet/v1/report", cfg.NodeID, fleet.Status{Version: "relocating", Protocol: fleet.ProtocolVersion}, &response); err != nil {
		return err
	}
	return fleet.WriteJSON(filepath.Join(dir, ConfigFile), cfg)
}
