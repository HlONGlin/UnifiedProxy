package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/control"
	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type encryptedTaskRuntime struct {
	seen fleet.Task
}

func (r *encryptedTaskRuntime) Snapshot() fleet.Status {
	return fleet.Status{Version: "test", Protocol: fleet.ProtocolVersion, Capabilities: []string{"cert-credentials"}}
}
func (r *encryptedTaskRuntime) Apply(context.Context, fleet.DesiredState) error { return nil }
func (r *encryptedTaskRuntime) Execute(_ context.Context, task fleet.Task) (json.RawMessage, error) {
	task.Payload = append(json.RawMessage(nil), task.Payload...)
	r.seen = task
	return json.RawMessage("{\"saved\":true}"), nil
}

func TestLoadConfigMigratesEncryptionIdentityAndPersistsItLocally(t *testing.T) {
	dir := t.TempDir()
	identity, err := fleet.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	legacy := Config{ControlURL: "http://127.0.0.1:2053", ControlKey: identity.Public(), NodeID: "legacy-node", Identity: identity}
	if err = fleet.WriteJSON(filepath.Join(dir, ConfigFile), legacy); err != nil {
		t.Fatal(err)
	}
	migrated, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Encryption.Public() == "" {
		t.Fatal("legacy agent did not create an X25519 encryption key")
	}
	reloaded, err := LoadConfig(dir)
	if err != nil || reloaded.Encryption.Public() != migrated.Encryption.Public() {
		t.Fatalf("migrated encryption key was not persisted: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil || !bytes.Contains(data, []byte(migrated.Encryption.PrivateKey)) {
		t.Fatalf("private encryption key was not saved in the child agent config: %v", err)
	}
}

func TestEncryptedCredentialTaskIsDecryptedOnlyForAgentRuntime(t *testing.T) {
	store, err := control.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(&control.Server{Store: store})
	defer server.Close()
	token, err := store.IssueToken("secret-node", "")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfg, err := Enroll(context.Background(), dir, server.URL, store.Identity.Public(), token, "secret-node")
	if err != nil {
		t.Fatal(err)
	}
	runtime := &encryptedTaskRuntime{}
	a, err := New(dir, runtime)
	if err != nil {
		t.Fatal(err)
	}
	taskID := fleet.ID()
	plaintext := []byte("{\"cloudflareToken\":\"only-on-child\",\"cloudflareAccountId\":\"account-id\",\"cloudflareZoneId\":\"zone-id\"}")
	ciphertext, err := fleet.EncryptTaskPayload(cfg.Encryption.Public(), cfg.NodeID, taskID, "cert-credentials", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	task := fleet.Task{ID: taskID, NodeID: cfg.NodeID, Kind: "cert-credentials", IdempotencyKey: taskID, Payload: ciphertext, Encrypted: true, ExpiresAt: time.Now().Add(time.Minute)}
	result := a.Process(context.Background(), task)
	if result.Status != "success" {
		t.Fatalf("encrypted task failed: %#v", result)
	}
	if string(runtime.seen.Payload) != string(plaintext) || !runtime.seen.Encrypted {
		t.Fatal("runtime did not receive decrypted credentials and encrypted-task marker")
	}
	receipt := a.journal.Receipts[taskID]
	if receipt == nil || !receipt.Encrypted || receipt.Payload != string(ciphertext) || bytes.Contains([]byte(receipt.Payload), plaintext) {
		t.Fatal("agent receipt persisted plaintext credentials")
	}
	if bytes.Contains(result.Data, plaintext) {
		t.Fatal("task result exposed plaintext credentials")
	}
}
