package fleet

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestFleetSigningRejectsReplayTamperAndWrongAudience(t *testing.T) {
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	e, err := Sign(id, "node", "POST /fleet/v1/poll", map[string]string{"test": "value"})
	if err != nil {
		t.Fatal(err)
	}
	var guard ReplayGuard
	if err = guard.Verify(e, id.Public(), "node", "POST /fleet/v1/poll", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err = guard.Verify(e, id.Public(), "node", "POST /fleet/v1/poll", time.Now()); err == nil {
		t.Fatal("replay accepted")
	}
	var fresh ReplayGuard
	if err = fresh.Verify(e, id.Public(), "node", "POST /fleet/v1/result", time.Now()); err == nil {
		t.Fatal("wrong audience accepted")
	}
	e.Payload = json.RawMessage(`{"test":"changed"}`)
	if err = fresh.Verify(e, id.Public(), "node", "POST /fleet/v1/poll", time.Now()); err == nil {
		t.Fatal("tampering accepted")
	}
	e, err = Sign(id, "node", "poll", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = fresh.Verify(e, id.Public(), "node", "poll", time.Now().Add(6*time.Minute)); err == nil {
		t.Fatal("expired envelope accepted")
	}
}
func TestFleetIdentityAndDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.json")
	a, err := LoadIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadIdentity(path)
	if err != nil || a.Public() != b.Public() {
		t.Fatal("identity changed", err)
	}
	s := DesiredState{NodeID: "node", Revision: 1, Metering: "agent", MihomoYAML: "mode: rule", Model: json.RawMessage(`{}`)}
	s.Digest, _ = s.ContentDigest()
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Revision = 2
	if err := s.Validate(); err != nil {
		t.Fatal("revision must not alter content digest")
	}
	s.MihomoYAML = "bad"
	if s.Validate() == nil {
		t.Fatal("modified YAML accepted")
	}
}
func TestFleetControlURL(t *testing.T) {
	for _, u := range []string{"http://example.com", "https://user:secret@example.com", "file:///tmp/controller", "https://example.com?secret=x"} {
		if ValidateControlURL(u) == nil {
			t.Errorf("accepted %s", u)
		}
	}
	for _, u := range []string{"https://example.com", "http://127.0.0.1:9000", "http://[::1]:9000"} {
		if err := ValidateControlURL(u); err != nil {
			t.Error(err)
		}
	}
}
