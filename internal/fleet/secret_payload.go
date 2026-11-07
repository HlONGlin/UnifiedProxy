package fleet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// EncryptionIdentity is a node-specific X25519 key. It is separate from the
// Ed25519 identity used to authenticate Fleet messages.
type EncryptionIdentity struct {
	PrivateKey string `json:"privateKey"`
}

func NewEncryptionIdentity() (EncryptionIdentity, error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return EncryptionIdentity{}, err
	}
	return EncryptionIdentity{PrivateKey: base64.StdEncoding.EncodeToString(key.Bytes())}, nil
}

func (i EncryptionIdentity) Key() (*ecdh.PrivateKey, error) {
	private, err := base64.StdEncoding.DecodeString(i.PrivateKey)
	if err != nil || len(private) != 32 {
		return nil, errors.New("invalid X25519 encryption identity")
	}
	return ecdh.X25519().NewPrivateKey(private)
}

func (i EncryptionIdentity) Public() string {
	key, err := i.Key()
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(key.PublicKey().Bytes())
}

func ValidateEncryptionPublicKey(encoded string) error {
	publicBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(publicBytes) != 32 {
		return errors.New("invalid X25519 encryption public key")
	}
	public, err := ecdh.X25519().NewPublicKey(publicBytes)
	if err != nil {
		return errors.New("invalid X25519 encryption public key")
	}
	probe, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if _, err = probe.ECDH(public); err != nil {
		return errors.New("invalid X25519 encryption public key")
	}
	return nil
}

// EncryptedPayload is safe to persist and back up: the plaintext is only
// recoverable by the selected node's X25519 private key.
type EncryptedPayload struct {
	Version            int    `json:"version"`
	EphemeralPublicKey string `json:"ephemeralPublicKey"`
	Nonce              string `json:"nonce"`
	Ciphertext         string `json:"ciphertext"`
}

func taskSecretAAD(nodeID, taskID, kind string) []byte {
	return []byte("UnifiedProxy-fleet-secret-v1\n" + nodeID + "\n" + taskID + "\n" + kind)
}

func taskSecretKey(shared []byte, nodeID, taskID, kind string) ([]byte, error) {
	salt := sha256.Sum256([]byte("UnifiedProxy-fleet-secret-salt-v1\n" + nodeID))
	info := "UnifiedProxy-fleet-secret-key-v1\n" + taskID + "\n" + kind
	return hkdf.Key(sha256.New, shared, salt[:], info, 32)
}

// EncryptTaskPayload seals data for a single node and binds it to the exact
// task identity, so the ciphertext cannot be moved to another node or task.
func EncryptTaskPayload(publicKey, nodeID, taskID, kind string, plaintext []byte) (json.RawMessage, error) {
	if !SafeName(nodeID) || !SafeName(taskID) || kind == "" || len(plaintext) == 0 || len(plaintext) > MaxMessage {
		return nil, errors.New("invalid encrypted task payload")
	}
	if err := ValidateEncryptionPublicKey(publicKey); err != nil {
		return nil, err
	}
	encodedPublic, _ := base64.StdEncoding.DecodeString(publicKey)
	recipient, _ := ecdh.X25519().NewPublicKey(encodedPublic)
	ephemeral, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	shared, err := ephemeral.ECDH(recipient)
	if err != nil {
		return nil, errors.New("could not establish encrypted task key")
	}
	defer clear(shared)
	key, err := taskSecretKey(shared, nodeID, taskID, kind)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := aead.Seal(nil, nonce, plaintext, taskSecretAAD(nodeID, taskID, kind))
	envelope := EncryptedPayload{
		Version:            1,
		EphemeralPublicKey: base64.StdEncoding.EncodeToString(ephemeral.PublicKey().Bytes()),
		Nonce:              base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:         base64.StdEncoding.EncodeToString(sealed),
	}
	data, err := json.Marshal(envelope)
	return data, err
}

func ValidateEncryptedTaskPayload(data []byte) error {
	var envelope EncryptedPayload
	if err := json.Unmarshal(data, &envelope); err != nil || envelope.Version != 1 {
		return errors.New("invalid encrypted task payload")
	}
	public, err := base64.StdEncoding.DecodeString(envelope.EphemeralPublicKey)
	if err != nil || len(public) != 32 {
		return errors.New("invalid encrypted task payload")
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil || len(nonce) != 12 {
		return errors.New("invalid encrypted task payload")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil || len(ciphertext) < 16 || len(ciphertext) > MaxMessage {
		return errors.New("invalid encrypted task payload")
	}
	if _, err = ecdh.X25519().NewPublicKey(public); err != nil {
		return errors.New("invalid encrypted task payload")
	}
	return nil
}

// DecryptTaskPayload is called only by the addressed Agent after it verifies
// the controller's signed task response.
func DecryptTaskPayload(identity EncryptionIdentity, nodeID, taskID, kind string, data []byte) ([]byte, error) {
	if err := ValidateEncryptedTaskPayload(data); err != nil {
		return nil, err
	}
	var envelope EncryptedPayload
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, errors.New("invalid encrypted task payload")
	}
	private, err := identity.Key()
	if err != nil {
		return nil, err
	}
	publicBytes, _ := base64.StdEncoding.DecodeString(envelope.EphemeralPublicKey)
	public, err := ecdh.X25519().NewPublicKey(publicBytes)
	if err != nil {
		return nil, errors.New("invalid encrypted task key")
	}
	shared, err := private.ECDH(public)
	if err != nil {
		return nil, errors.New("encrypted task key does not match this node")
	}
	defer clear(shared)
	key, err := taskSecretKey(shared, nodeID, taskID, kind)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, _ := base64.StdEncoding.DecodeString(envelope.Nonce)
	ciphertext, _ := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	plaintext, err := aead.Open(nil, nonce, ciphertext, taskSecretAAD(nodeID, taskID, kind))
	if err != nil {
		return nil, fmt.Errorf("encrypted task authentication failed")
	}
	return plaintext, nil
}
