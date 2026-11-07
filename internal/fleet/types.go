// Package fleet defines the versioned control/node protocol.
package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const ProtocolVersion = 1
const MaxMessage = 16 << 20

const HeartbeatInterval = 15 * time.Second
const HeartbeatTimeout = 90 * time.Second

// Heartbeat deliberately carries no runtime snapshot: maintenance may hold the
// runtime lock for minutes, and an old snapshot must not overwrite newer data.
type Heartbeat struct {
	PublicIP string `json:"publicIP,omitempty"`
}

var ErrUpgradePending = errors.New("verified upgrade is awaiting restart and health verification")

var TaskKinds = []string{"apply-state", "collect-status", "test-egress", "kernel-switch", "geo-update", "cert-issue", "cert-renew", "cert-bind", "cert-credentials", "agent-upgrade", "purge", "core-control", "traffic-reset", "vpngate-install", "vpngate-reconnect", "outbound-delay"}

type DesiredState struct {
	NodeID       string          `json:"nodeId"`
	Revision     uint64          `json:"revision"`
	Digest       string          `json:"digest"`
	MihomoYAML   string          `json:"mihomoYAML"`
	Model        json.RawMessage `json:"model"`
	Certificates []Certificate   `json:"certificates,omitempty"`
	Sidecars     []Sidecar       `json:"sidecars,omitempty"`
	Metering     string          `json:"quotaEnforcedAt"`
}

type Certificate struct {
	Name        string `json:"name"`
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"privateKey"`
}

// Sidecar commands are restricted to locally provisioned executable names.
// Configuration is data; commands never pass through a shell.
type Sidecar struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	Config          string `json:"config"`
	Enabled         bool   `json:"enabled"`
	Address         string `json:"address,omitempty"`
	Port            int    `json:"port,omitempty"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	ServerName      string `json:"serverName,omitempty"`
	BackendPassword string `json:"backendPassword,omitempty"`
	Cipher          string `json:"cipher,omitempty"`
	Metering        string `json:"metering"`
}

func (s DesiredState) ContentDigest() (string, error) {
	s.Digest = ""
	s.Revision = 0
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

func (s DesiredState) Validate() error {
	if s.NodeID == "" || s.Revision == 0 || s.Metering != "agent" || !json.Valid(s.Model) || s.MihomoYAML == "" {
		return fmt.Errorf("invalid desired state")
	}
	d, err := s.ContentDigest()
	if err != nil || d != s.Digest {
		return fmt.Errorf("desired state digest mismatch")
	}
	return nil
}

type Task struct {
	ID             string          `json:"id"`
	NodeID         string          `json:"nodeId"`
	Kind           string          `json:"kind"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	Encrypted      bool            `json:"encrypted,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	ExpiresAt      time.Time       `json:"expiresAt"`
	LeaseUntil     time.Time       `json:"leaseUntil,omitempty"`
	Attempts       int             `json:"attempts"`
	Status         string          `json:"status"`
	Result         *Result         `json:"result,omitempty"`
}

type Result struct {
	TaskID string          `json:"taskId"`
	Status string          `json:"status"`
	Error  string          `json:"error,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type Status struct {
	PublicIP            string           `json:"publicIP,omitempty"`
	Version             string           `json:"version"`
	Protocol            int              `json:"protocol"`
	Capabilities        []string         `json:"capabilities"`
	OS                  string           `json:"os"`
	Arch                string           `json:"arch"`
	Running             bool             `json:"running"`
	Revision            uint64           `json:"revision"`
	Digest              string           `json:"digest"`
	EncryptionPublicKey string           `json:"encryptionPublicKey,omitempty"`
	Traffic             map[string]Usage `json:"traffic,omitempty"`
	Details             json.RawMessage  `json:"details,omitempty"`
	Error               string           `json:"error,omitempty"`
}

type Usage struct {
	LegacyCombined        uint64 `json:"legacyCombined,omitempty"`
	AllTimeLegacyCombined uint64 `json:"allTimeLegacyCombined,omitempty"`
	Up                    uint64 `json:"up"`
	Down                  uint64 `json:"down"`
	AllTimeUp             uint64 `json:"allTimeUp"`
	AllTimeDown           uint64 `json:"allTimeDown"`
	ExpiryTime            int64  `json:"expiryTime,omitempty"`
	RenewCount            int    `json:"renewCount,omitempty"`
	FirstOnline           string `json:"firstOnline,omitempty"`
	LastTrafficReset      string `json:"lastTrafficReset,omitempty"`
	LastOnline            string `json:"lastOnline,omitempty"`
}

type Poll struct {
	Status Status `json:"status"`
	Wait   int    `json:"wait"`
}
type PollResponse struct {
	Tasks    []Task `json:"tasks"`
	Revision uint64 `json:"revision"`
	Digest   string `json:"digest"`
}
type Enrollment struct {
	Token               string `json:"token"`
	PublicKey           string `json:"publicKey"`
	EncryptionPublicKey string `json:"encryptionPublicKey,omitempty"`
	Name                string `json:"name"`
}
type EnrollmentResponse struct {
	NodeID              string `json:"nodeId"`
	PublicKey           string `json:"publicKey"`
	EncryptionPublicKey string `json:"encryptionPublicKey,omitempty"`
	Protocol            int    `json:"protocol"`
}

func Supports(capabilities []string, kind string) bool {
	for _, c := range capabilities {
		if c == kind {
			return true
		}
	}
	return false
}

func RequiresEncryptedTaskPayload(kind string) bool {
	return kind == "cert-credentials"
}
