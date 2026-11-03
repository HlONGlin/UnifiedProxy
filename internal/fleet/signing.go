package fleet

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Identity struct {
	PrivateKey string `json:"privateKey"`
}

func NewIdentity() (Identity, error) {
	_, k, err := ed25519.GenerateKey(rand.Reader)
	return Identity{PrivateKey: base64.StdEncoding.EncodeToString(k)}, err
}
func (i Identity) Key() (ed25519.PrivateKey, error) {
	k, err := base64.StdEncoding.DecodeString(i.PrivateKey)
	if err != nil || len(k) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 identity")
	}
	return ed25519.PrivateKey(k), nil
}
func (i Identity) Public() string {
	k, err := i.Key()
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(k.Public().(ed25519.PublicKey))
}
func LoadIdentity(path string) (Identity, error) {
	var i Identity
	err := ReadJSON(path, &i)
	if errors.Is(err, os.ErrNotExist) {
		i, err = NewIdentity()
		if err == nil {
			err = WriteJSON(path, i)
		}
	}
	if err != nil {
		return i, err
	}
	_, err = i.Key()
	return i, err
}
func LoadIdentityWithLegacy(path, legacy string) (Identity, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		var identity Identity
		if err = ReadJSON(legacy, &identity); err == nil {
			if _, err = identity.Key(); err != nil {
				return identity, err
			}
			if err = WriteJSON(path, identity); err != nil {
				return identity, err
			}
			return identity, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return identity, err
		}
	}
	return LoadIdentity(path)
}
func ID() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func Fingerprint(public string) string {
	b, _ := base64.StdEncoding.DecodeString(public)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

type Envelope struct {
	Version   int             `json:"version"`
	Sender    string          `json:"sender"`
	Audience  string          `json:"audience"`
	Timestamp int64           `json:"timestamp"`
	Nonce     string          `json:"nonce"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

func (e Envelope) signingBytes() []byte {
	h := sha256.Sum256(e.Payload)
	return []byte(strconv.Itoa(e.Version) + "\n" + e.Sender + "\n" + e.Audience + "\n" + strconv.FormatInt(e.Timestamp, 10) + "\n" + e.Nonce + "\n" + hex.EncodeToString(h[:]))
}
func Sign(identity Identity, sender, audience string, payload any) (Envelope, error) {
	e := Envelope{Version: ProtocolVersion, Sender: sender, Audience: audience, Timestamp: time.Now().Unix(), Nonce: ID()}
	b, err := json.Marshal(payload)
	if err != nil {
		return e, err
	}
	e.Payload = b
	k, err := identity.Key()
	if err != nil {
		return e, err
	}
	e.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(k, e.signingBytes()))
	return e, nil
}

type ReplayGuard struct {
	File string
	mu   sync.Mutex
	seen map[string]int64
}

func (g *ReplayGuard) Verify(e Envelope, public, sender, audience string, now time.Time) error {
	if e.Version != ProtocolVersion || e.Sender != sender || e.Audience != audience || len(e.Nonce) != 48 || !json.Valid(e.Payload) {
		return errors.New("invalid signed envelope")
	}
	if e.Timestamp < now.Unix()-300 || e.Timestamp > now.Unix()+300 {
		return errors.New("signature timestamp outside five-minute window")
	}
	pub, err := base64.StdEncoding.DecodeString(public)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return errors.New("invalid public key")
	}
	sig, err := base64.StdEncoding.DecodeString(e.Signature)
	if err != nil || !ed25519.Verify(pub, e.signingBytes(), sig) {
		return errors.New("invalid signature")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.seen == nil {
		loaded := map[string]int64{}
		if g.File != "" {
			if err := ReadJSON(g.File, &loaded); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("load replay cache: %w", err)
			}
		}
		if loaded == nil {
			return errors.New("invalid replay cache")
		}
		g.seen = loaded
	}
	for k, v := range g.seen {
		if v < now.Unix()-600 {
			delete(g.seen, k)
		}
	}
	k := sender + ":" + e.Nonce
	if _, exists := g.seen[k]; exists {
		return errors.New("replayed message")
	}
	if len(g.seen) >= 100000 {
		return errors.New("replay cache full")
	}
	g.seen[k] = e.Timestamp
	if g.File != "" {
		if err := WriteJSON(g.File, g.seen); err != nil {
			delete(g.seen, k)
			return fmt.Errorf("persist replay protection: %w", err)
		}
	}
	return nil
}

func ValidateControlURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid control URL")
	}
	if u.Scheme == "https" {
		return nil
	}
	h := u.Hostname()
	ip := net.ParseIP(h)
	if u.Scheme == "http" && (h == "localhost" || (ip != nil && ip.IsLoopback())) {
		return nil
	}
	return fmt.Errorf("control URL requires HTTPS (HTTP is allowed only on loopback)")
}

func SafeName(name string) bool {
	if name == "" || len(name) > 128 || name == "." || name == ".." {
		return false
	}
	return !strings.ContainsAny(name, "/\\:\x00\r\n")
}
