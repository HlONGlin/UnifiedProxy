package fleet

import (
	"bytes"
	"testing"
)

func TestEncryptedTaskPayloadIsNodeAndTaskBound(t *testing.T) {
	identity, err := NewEncryptionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("{\"cloudflareToken\":\"secret-token\",\"cloudflareAccountId\":\"account\",\"cloudflareZoneId\":\"zone\"}")
	taskID := ID()
	envelope, err := EncryptTaskPayload(identity.Public(), "node-1", taskID, "cert-credentials", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(envelope, plaintext) || bytes.Contains(envelope, []byte("secret-token")) {
		t.Fatal("encrypted payload contains plaintext credentials")
	}
	if err = ValidateEncryptedTaskPayload(envelope); err != nil {
		t.Fatal(err)
	}
	opened, err := DecryptTaskPayload(identity, "node-1", taskID, "cert-credentials", envelope)
	if err != nil || !bytes.Equal(opened, plaintext) {
		t.Fatalf("encrypted payload did not round trip: %s, %v", opened, err)
	}
	for _, binding := range []struct{ nodeID, taskID, kind string }{
		{"node-2", taskID, "cert-credentials"},
		{"node-1", ID(), "cert-credentials"},
		{"node-1", taskID, "cert-issue"},
	} {
		if _, err = DecryptTaskPayload(identity, binding.nodeID, binding.taskID, binding.kind, envelope); err == nil {
			t.Fatalf("payload decrypted with mismatched binding %#v", binding)
		}
	}
	other, err := NewEncryptionIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = DecryptTaskPayload(other, "node-1", taskID, "cert-credentials", envelope); err == nil {
		t.Fatal("another node decrypted the encrypted payload")
	}
}
