package control

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

func TestPresenceAddressPriorityPersistenceAndRebind(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := enrollTest(t, s)
	if n.Online(time.Now()) || !n.LastSeen.IsZero() {
		t.Fatal("enrollment is not a heartbeat")
	}
	if err := s.UpdateStatus(n.ID, fleet.Status{PublicIP: "8.8.8.8", Revision: 7}, n.PublicKey); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateNode(n.ID, "manual", "proxy.example"); err != nil {
		t.Fatal(err)
	}
	if err := s.Heartbeat(n.ID, "1.1.1.1", n.PublicKey); err != nil {
		t.Fatal(err)
	}
	n = s.Node(n.ID)
	if n.EffectiveAddress() != "proxy.example" || n.SeenIP != "1.1.1.1" || n.Status.Revision != 7 {
		t.Fatal("manual priority or runtime status lost", n)
	}
	if err := s.UpdateNode(n.ID, "auto", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Heartbeat(n.ID, "127.0.0.1", n.PublicKey); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	n = s.Node(n.ID)
	if n.EffectiveAddress() != "1.1.1.1" || n.AddressSource() != "auto" {
		t.Fatal("automatic address not persisted", n)
	}
	if !n.Online(n.LastSeen.Add(fleet.HeartbeatTimeout-time.Nanosecond)) || n.Online(n.LastSeen.Add(fleet.HeartbeatTimeout)) {
		t.Fatal("offline timeout boundary")
	}
	before := n.LastSeen
	if s.Heartbeat(n.ID, "9.9.9.9", "wrong-key") == nil || !s.Node(n.ID).LastSeen.Equal(before) {
		t.Fatal("wrong identity changed presence")
	}
	token, _ := s.IssueToken("rebind", n.ID)
	id, _ := fleet.NewIdentity()
	n, err = s.Enroll(fleet.Enrollment{Token: token, PublicKey: id.Public()})
	if err != nil {
		t.Fatal(err)
	}
	if n.SeenIP != "" || !n.LastSeen.IsZero() || n.Online(time.Now()) {
		t.Fatal("rebind retained old presence")
	}
	if err := s.Revoke(n.ID); err != nil {
		t.Fatal(err)
	}
	if s.Heartbeat(n.ID, "8.8.8.8", n.PublicKey) == nil {
		t.Fatal("revoked heartbeat accepted")
	}
}

func TestSignedHeartbeatIgnoresProxyHeadersAndRejectsReplay(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id, _ := fleet.NewIdentity()
	token, _ := s.IssueToken("auto", "")
	n, err := s.Enroll(fleet.Enrollment{Token: token, PublicKey: id.Public()})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: s}
	send := func(envelope fleet.Envelope, remote string) int {
		b, _ := json.Marshal(envelope)
		r := httptest.NewRequest("POST", "/fleet/v1/heartbeat", bytes.NewReader(b))
		r.RemoteAddr = remote
		r.Header.Set("X-Forwarded-For", "9.9.9.9")
		r.Header.Set("X-Real-IP", "9.9.9.9")
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w.Code
	}
	e, _ := fleet.Sign(id, n.ID, "POST /fleet/v1/heartbeat", fleet.Heartbeat{})
	if send(e, "127.0.0.1:443") != 200 || s.Node(n.ID).SeenIP != "" {
		t.Fatal("trusted proxy header")
	}
	if send(e, "8.8.8.8:443") != 401 || s.Node(n.ID).SeenIP != "" {
		t.Fatal("replay changed address")
	}
	e, _ = fleet.Sign(id, n.ID, "POST /fleet/v1/heartbeat", fleet.Heartbeat{})
	if send(e, "8.8.8.8:443") != 200 || s.Node(n.ID).SeenIP != "8.8.8.8" {
		t.Fatal("socket peer fallback failed")
	}
	e, _ = fleet.Sign(id, n.ID, "POST /fleet/v1/heartbeat", fleet.Heartbeat{PublicIP: "1.1.1.1"})
	if send(e, "8.8.8.8:443") != 200 || s.Node(n.ID).SeenIP != "1.1.1.1" {
		t.Fatal("reported IP should take priority")
	}
	e.Signature = "invalid"
	if send(e, "9.9.9.9:443") != 401 || s.Node(n.ID).SeenIP != "1.1.1.1" {
		t.Fatal("unsigned update accepted")
	}
}
