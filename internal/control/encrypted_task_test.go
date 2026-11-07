package control

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestEncryptedCredentialTasksStayCiphertextInStoreAndBackup(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	node, err := store.AddNode("edge", "")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := fleet.NewEncryptionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	status := fleet.Status{Version: "test", Protocol: fleet.ProtocolVersion, Capabilities: []string{"cert-credentials"}, EncryptionPublicKey: identity.Public()}
	if err = store.UpdateStatus(node.ID, status); err != nil {
		t.Fatal(err)
	}
	taskID := fleet.ID()
	plaintext := []byte("{\"cloudflareToken\":\"never-store-this-token\",\"cloudflareAccountId\":\"account-secret\",\"cloudflareZoneId\":\"zone-secret\"}")
	payload, err := fleet.EncryptTaskPayload(identity.Public(), node.ID, taskID, "cert-credentials", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Enqueue(node.ID, "cert-credentials", "plain", plaintext, time.Minute); err == nil {
		t.Fatal("unencrypted credential task was accepted")
	}
	task, err := store.EnqueueEncrypted(node.ID, taskID, "cert-credentials", payload, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !task.Encrypted || bytes.Contains(task.Payload, plaintext) {
		t.Fatal("credential task was not stored as ciphertext")
	}
	visible := store.Tasks()
	if len(visible) != 1 || !visible[0].Encrypted || len(visible[0].Payload) != 0 {
		t.Fatal("encrypted payload was exposed in task history")
	}
	for label, content := range map[string][]byte{
		"state":  func() []byte { data, _ := os.ReadFile(store.path); return data }(),
		"backup": func() []byte { data, _ := store.Backup(); return data }(),
	} {
		if bytes.Contains(content, plaintext) || bytes.Contains(content, []byte("never-store-this-token")) || bytes.Contains(content, []byte("cloudflareToken")) {
			t.Fatalf("%s contains plaintext Cloudflare credentials", label)
		}
	}
	claimed, err := store.Claim(node.ID, time.Now())
	if err != nil || len(claimed) != 1 || !claimed[0].Encrypted || !json.Valid(claimed[0].Payload) {
		t.Fatalf("agent did not receive the encrypted task: %#v, %v", claimed, err)
	}
	opened, err := fleet.DecryptTaskPayload(identity, node.ID, claimed[0].ID, claimed[0].Kind, claimed[0].Payload)
	if err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("claimed payload was not decryptable on its target node: %s, %v", opened, err)
	}
}

func TestEncryptionKeyRotationCancelsLeasedCredentialTasks(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	node, err := store.AddNode("edge", "")
	if err != nil {
		t.Fatal(err)
	}
	oldIdentity, err := fleet.NewEncryptionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	status := fleet.Status{Version: "test", Protocol: fleet.ProtocolVersion, Capabilities: []string{"cert-credentials"}, EncryptionPublicKey: oldIdentity.Public()}
	if err = store.UpdateStatus(node.ID, status); err != nil {
		t.Fatal(err)
	}
	oldTaskID := fleet.ID()
	oldPayload, err := fleet.EncryptTaskPayload(oldIdentity.Public(), node.ID, oldTaskID, "cert-credentials", []byte("{\"cloudflareToken\":\"old-secret\"}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.EnqueueEncrypted(node.ID, oldTaskID, "cert-credentials", oldPayload, time.Minute); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.Claim(node.ID, time.Now())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("could not lease old-key task: %#v, %v", claimed, err)
	}

	newIdentity, err := fleet.NewEncryptionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	status.EncryptionPublicKey = newIdentity.Public()
	if err = store.UpdateStatus(node.ID, status); err != nil {
		t.Fatal(err)
	}
	tasks := store.Tasks()
	if len(tasks) != 1 || tasks[0].ID != oldTaskID || tasks[0].Status != "cancelled" {
		t.Fatalf("old-key task was not cancelled: %#v", tasks)
	}

	newTaskID := fleet.ID()
	newPayload, err := fleet.EncryptTaskPayload(newIdentity.Public(), node.ID, newTaskID, "cert-credentials", []byte("{\"cloudflareToken\":\"new-secret\"}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.EnqueueEncrypted(node.ID, newTaskID, "cert-credentials", newPayload, time.Minute); err != nil {
		t.Fatalf("new-key task was rejected: %v", err)
	}
	claimed, err = store.Claim(node.ID, time.Now())
	if err != nil || len(claimed) != 1 || claimed[0].ID != newTaskID {
		t.Fatalf("new-key task was not delivered: %#v, %v", claimed, err)
	}
	opened, err := fleet.DecryptTaskPayload(newIdentity, node.ID, newTaskID, "cert-credentials", claimed[0].Payload)
	if err != nil || string(opened) != "{\"cloudflareToken\":\"new-secret\"}" {
		t.Fatalf("new-key task did not decrypt: %s, %v", opened, err)
	}
}
