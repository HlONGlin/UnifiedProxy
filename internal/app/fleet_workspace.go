package app

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/hlonglin/unifiedproxy/internal/fleet"
)

// The URL owns the server selection. Cookies still belong to the single control
// login, and neither a stale browser selection nor a supplied header can move a
// request from this workspace to another server.
func (a *App) withFleetWorkspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.fleet == nil {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/fleet/" || r.URL.Path == "/fleet" {
			http.Redirect(w, r, a.manager.panelPath(), http.StatusFound)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		workspace, id, rest := "", "", ""
		if path == "host" || strings.HasPrefix(path, "host/") {
			workspace, rest = "host", strings.TrimPrefix(path, "host")
		} else if strings.HasPrefix(path, "nodes/") {
			parts := strings.SplitN(strings.TrimPrefix(path, "nodes/"), "/", 2)
			id = parts[0]
			if !fleet.SafeName(id) {
				http.NotFound(w, r)
				return
			}
			workspace = "node"
			if len(parts) == 2 {
				rest = "/" + parts[1]
			}
		}
		if workspace == "" {
			next.ServeHTTP(w, r)
			return
		}
		destination := "host/"
		if id != "" {
			destination = "nodes/" + id + "/"
		}
		if rest == "" {
			http.Redirect(w, r, a.manager.panelPath()+destination, http.StatusPermanentRedirect)
			return
		}
		if rest == "/login" || (rest == "/" && !a.isLoggedIn(r)) {
			http.Redirect(w, r, a.manager.panelPath()+"login?next="+url.QueryEscape(destination), http.StatusFound)
			return
		}
		if id != "" && (rest == "/" || strings.HasPrefix(rest, "/api/")) {
			if !a.isLoggedIn(r) {
				writeJSON(w, 401, apiResponse{Message: "unauthorized"})
				return
			}
			n := a.fleet.store.Node(id)
			if n == nil || n.Revoked {
				http.NotFound(w, r)
				return
			}
		}
		if rest == "/" {
			a.serveWorkspacePage(w, "web/index.html", workspace, id)
			return
		}
		r = r.Clone(r.Context())
		r.URL.Path = rest
		r.URL.RawPath = ""
		r.Header.Del("X-Fleet-Node")
		if id != "" && strings.HasPrefix(rest, "/api/") {
			// Templates and release metadata are explicitly shared control resources.
			shared := rest == "/api/language" || rest == "/api/auth/logout" || rest == "/api/core/releases" || rest == "/api/sub-templates" || strings.HasPrefix(rest, "/api/sub-templates/")
			if !shared {
				if !a.remoteRoute(rest) {
					writeJSON(w, 404, apiResponse{Message: "not available in this server workspace"})
					return
				}
				r.Header.Set("X-Fleet-Node", id)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) loginDestination(r *http.Request) string {
	return a.workspaceDestination(r.URL.Query().Get("next"))
}

func (a *App) workspaceDestination(next string) string {
	base := a.manager.panelPath()
	if a.fleet == nil {
		return base
	}
	if next == "host/" {
		return base + next
	}
	parts := strings.Split(next, "/")
	if len(parts) == 3 && parts[0] == "nodes" && parts[2] == "" && fleet.SafeName(parts[1]) {
		return base + next
	}
	return base
}
