package app

import (
	"net/http/httptest"
	"strings"
	"testing"

	installers "github.com/hlonglin/unifiedproxy/scripts"
)

func TestFleetPublicInstallerHasNoEnrollmentSecrets(t *testing.T) {
	a := fleetTestApp(t)
	for _, method := range []string{"GET", "HEAD", "POST"} {
		w := httptest.NewRecorder()
		a.withFleetTransport(a.routes()).ServeHTTP(w, httptest.NewRequest(method, "/fleet/v1/install.sh?token=must-not-be-embedded", nil))
		if method == "POST" {
			if w.Code != 405 {
				t.Fatal(w.Code)
			}
			continue
		}
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal(w.Code, w.Header())
		}
		if method == "HEAD" {
			if w.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
			continue
		}
		if w.Body.String() != installers.NodeInstaller || strings.Contains(w.Body.String(), "must-not-be-embedded") || strings.Contains(w.Body.String(), a.fleet.store.Identity.Public()) {
			t.Fatal("installer contains per-request credentials")
		}
	}
	node := &App{role: "node"}
	w := httptest.NewRecorder()
	node.withFleetTransport(a.routes()).ServeHTTP(w, httptest.NewRequest("GET", "/fleet/v1/install.sh", nil))
	if w.Code != 404 {
		t.Fatal("node exposed a control bootstrap", w.Code)
	}
}
