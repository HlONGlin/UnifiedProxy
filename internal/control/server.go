package control

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

type Server struct {
	once   sync.Once
	Store  *Store
	replay fleet.ReplayGuard
}

func jsonResponse(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.once.Do(func() { s.replay.File = filepath.Join(filepath.Dir(s.Store.path), "fleet-replays.json") })
	if r.Method != "POST" {
		jsonResponse(w, 405, map[string]string{"error": "POST required"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, fleet.MaxMessage+1))
	if err != nil || len(data) > fleet.MaxMessage {
		jsonResponse(w, 413, map[string]string{"error": "message too large"})
		return
	}
	var e fleet.Envelope
	if err = json.Unmarshal(data, &e); err != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid envelope"})
		return
	}
	audience := "POST " + r.URL.Path
	if r.URL.Path == "/fleet/v1/enroll" {
		var enroll fleet.Enrollment
		if json.Unmarshal(e.Payload, &enroll) != nil || len(enroll.Name) > 200 {
			jsonResponse(w, 400, map[string]string{"error": "invalid enrollment"})
			return
		}
		if err = s.replay.Verify(e, enroll.PublicKey, fleet.Fingerprint(enroll.PublicKey), audience, time.Now()); err != nil {
			jsonResponse(w, 401, map[string]string{"error": err.Error()})
			return
		}
		n, err := s.Store.Enroll(enroll)
		if err != nil {
			jsonResponse(w, 403, map[string]string{"error": err.Error()})
			return
		}
		s.reply(w, e, fleet.EnrollmentResponse{NodeID: n.ID, PublicKey: s.Store.Identity.Public(), Protocol: fleet.ProtocolVersion})
		return
	}
	n := s.Store.Node(e.Sender)
	if n == nil || n.Revoked || s.replay.Verify(e, n.PublicKey, n.ID, audience, time.Now()) != nil {
		jsonResponse(w, 401, map[string]string{"error": "invalid or revoked identity, expired signature, or replay"})
		return
	}
	switch strings.TrimPrefix(r.URL.Path, "/fleet/v1/") {
	case "heartbeat":
		var heartbeat fleet.Heartbeat
		if err = json.Unmarshal(e.Payload, &heartbeat); err == nil {
			err = s.Store.Heartbeat(n.ID, observedPublicIP(heartbeat.PublicIP, r.RemoteAddr), n.PublicKey)
		}
		if err != nil {
			jsonResponse(w, 400, map[string]string{"error": err.Error()})
			return
		}
		s.reply(w, e, map[string]bool{"ok": true})
	case "poll":
		var p fleet.Poll
		if json.Unmarshal(e.Payload, &p) != nil || p.Wait != 0 {
			jsonResponse(w, 400, map[string]string{"error": "invalid poll"})
			return
		}
		p.Status.PublicIP = observedPublicIP(p.Status.PublicIP, r.RemoteAddr)
		if err = s.Store.UpdateStatus(n.ID, p.Status, n.PublicKey); err != nil {
			jsonResponse(w, 400, map[string]string{"error": err.Error()})
			return
		}
		tasks, err := s.Store.Claim(n.ID, time.Now(), n.PublicKey)
		if err != nil {
			jsonResponse(w, 401, map[string]string{"error": err.Error()})
			return
		}
		s.pollReply(w, e, n.ID, n.PublicKey, tasks)
	case "report":
		var status fleet.Status
		if err = json.Unmarshal(e.Payload, &status); err == nil {
			status.PublicIP = observedPublicIP(status.PublicIP, r.RemoteAddr)
			err = s.Store.UpdateStatus(n.ID, status, n.PublicKey)
		}
		if err != nil {
			jsonResponse(w, 400, map[string]string{"error": err.Error()})
			return
		}
		s.reply(w, e, map[string]bool{"ok": true})
	case "result":
		var result fleet.Result
		if err = json.Unmarshal(e.Payload, &result); err == nil {
			err = s.Store.Result(n.ID, result, n.PublicKey)
		}
		if err != nil {
			jsonResponse(w, 400, map[string]string{"error": err.Error()})
			return
		}
		s.reply(w, e, map[string]bool{"ok": true})
	default:
		jsonResponse(w, 404, map[string]string{"error": "unknown fleet endpoint"})
	}
}

func observedPublicIP(reported, remote string) string {
	if ip := fleet.PublicIP(reported); ip != "" {
		return ip
	}
	// Proxy headers are untrusted. Private reverse-proxy peers remain unresolved.
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return ""
	}
	return fleet.PublicIP(host)
}
func (s *Server) reply(w http.ResponseWriter, request fleet.Envelope, value any) {
	e, err := fleet.Sign(s.Store.Identity, "control", "response:"+request.Nonce, value)
	if err != nil {
		jsonResponse(w, 500, map[string]string{"error": "signing failed"})
		return
	}
	jsonResponse(w, 200, e)
}
func (s *Server) pollReply(w http.ResponseWriter, e fleet.Envelope, id, key string, tasks []fleet.Task) {
	n := s.Store.Node(id)
	if checkIdentity(n, []string{key}) != nil {
		jsonResponse(w, 401, map[string]string{"error": errors.New("node identity changed").Error()})
		return
	}
	p := fleet.PollResponse{Tasks: tasks}
	if n.Desired != nil {
		p.Revision = n.Desired.Revision
		p.Digest = n.Desired.Digest
	}
	s.reply(w, e, p)
}
