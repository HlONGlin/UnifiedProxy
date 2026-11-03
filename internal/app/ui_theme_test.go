package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestNormalizeUITheme(t *testing.T) {
	for input, want := range map[string]string{"": "t0", " md3 ": "t0", "liquid": "t0", "shadcn": "t0", "t0": "t0", " t1 ": "t1", "t2": "t2", "t3": "t3", "t4": "t4", "T4": "t0", "glass": "t0", `t1"><script>`: "t0"} {
		if got := normalizeUITheme(input); got != want {
			t.Fatalf("normalizeUITheme(%q)=%q want %q", input, got, want)
		}
	}
}

// The new contract is one component system for all palettes and old backups.
func TestThemedPageKeepsOnlyTheChosenThemeAssets(t *testing.T) {
	for _, name := range []string{"web/index.html", "web/login.html"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, theme := range []string{"t0", "t1", "t2", "t3", "t4", "shadcn", "md3", "liquid", "bogus"} {
			out := themedPage(string(raw), theme)
			if !strings.Contains(out, `data-color-theme="`+normalizeUITheme(theme)+`"`) || !strings.Contains(out, `data-theme="light"`) {
				t.Fatalf("%s/%s: wrong axes", name, theme)
			}
			for _, asset := range []string{"themes.css", "components.css", "appearance.js"} {
				if !strings.Contains(out, "static/"+asset) {
					t.Fatalf("%s/%s missing %s", name, theme, asset)
				}
			}
			for _, old := range []string{"data-ui-only=", "theme-shadcn.css", "theme-md3.css", "liquid-optics.js", "liquid-glass-gl.js", "liquid-toggle.js", "liquid-button.js", "icons-material.js"} {
				if strings.Contains(out, old) {
					t.Fatalf("%s/%s loads old identity %s", name, theme, old)
				}
			}
		}
	}
}

func TestServedPagesUseTheSettingsTheme(t *testing.T) {
	app := i18nTestApp(t)
	for _, theme := range []string{"t0", "t1", "t2", "t3", "t4", "md3", "liquid"} {
		app.manager.state.Settings.UITheme = theme
		for _, page := range []string{"web/index.html", "web/login.html"} {
			recorder := httptest.NewRecorder()
			app.serveLocalizedPage(recorder, page)
			if !strings.Contains(recorder.Body.String(), `data-color-theme="`+normalizeUITheme(theme)+`"`) {
				t.Fatalf("%s theme %s not injected", page, theme)
			}
		}
	}
}

func TestUIThemeSettingValidationAndMigration(t *testing.T) {
	m := &CoreManager{dataDir: t.TempDir()}
	if err := m.load(); err != nil {
		t.Fatal(err)
	}
	if m.state.Settings.UITheme != "t0" {
		t.Fatalf("default=%q", m.state.Settings.UITheme)
	}
	settings := m.state.Settings
	settings.Password = ""
	for _, theme := range []string{"shadcn", "md3", "liquid", "t0", "t1", "t2", "t3", "t4", ""} {
		settings.UITheme = theme
		if err := m.updateSettings(settings); err != nil {
			t.Fatalf("legacy/new setting %s rejected: %v", theme, err)
		}
		if normalizeUITheme(m.state.Settings.UITheme) != normalizeUITheme(theme) {
			t.Fatalf("wrong palette for %s", theme)
		}
	}
	before := m.state.Settings
	settings.UITheme = "glass"
	if err := m.updateSettings(settings); err == nil {
		t.Fatal("unknown appearance accepted")
	}
	if m.state.Settings.UITheme != before.UITheme {
		t.Fatal("rejected update mutated appearance")
	}
	for _, legacy := range []string{"", "shadcn", "md3", "liquid"} {
		m.state.Settings.UITheme = legacy
		username := m.state.Settings.Username
		panelPath := m.state.Settings.PanelPath
		if err := m.saveLocked(); err != nil {
			t.Fatal(err)
		}
		reloaded := &CoreManager{dataDir: m.dataDir}
		if err := reloaded.load(); err != nil {
			t.Fatal(err)
		}
		if reloaded.state.Settings.UITheme != "t0" || reloaded.state.Settings.Username != username || reloaded.state.Settings.PanelPath != panelPath {
			t.Fatal("legacy appearance migration changed nonappearance settings")
		}
	}
}

func TestSubscriptionPageFollowsTheTheme(t *testing.T) {
	for _, theme := range []string{"t0", "t1", "t2", "t3", "t4", "shadcn", "md3", "liquid"} {
		settings := Settings{PanelPath: "/panel/", SubscriptionPath: "/sub", ClashPath: "/clash", UITheme: theme}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/sub/abc123def4567890", nil)
		snapshot := subscriptionSnapshot{Settings: settings, Token: "abc123def4567890", Username: "alice"}
		if err := renderSubscriptionPage(recorder, request, snapshot, nil); err != nil {
			t.Fatal(err)
		}
		body := recorder.Body.String()
		for _, want := range []string{`data-color-theme="` + normalizeUITheme(theme) + `"`, `href="/sub/_assets/themes.css"`, `href="/sub/_assets/components.css"`, `src="/sub/_assets/appearance.js"`} {
			if !strings.Contains(body, want) {
				t.Fatalf("subscription/%s missing %s", theme, want)
			}
		}
		if strings.Contains(body, "liquid-toggle") || strings.Contains(body, "subscription-themes.css") {
			t.Fatal("subscription serves retired skins")
		}
	}
}

// Preserve explicit MIME and path containment checks on public assets.
func TestSubscriptionThemeAssetsArePublic(t *testing.T) {
	app := &App{manager: &CoreManager{dataDir: t.TempDir(), state: State{Settings: Settings{PanelPath: "/panel/", SubscriptionPath: "/sub", ClashPath: "/clash"}}}, session: "test-session"}
	for path, typ := range map[string]string{
		"themes.css": "text/css", "components.css": "text/css", "subscription.css": "text/css", "appearance.js": "text/javascript", "fonts.css": "text/css",
		"fonts/geist-latin-wght-normal.woff2": "font/woff2", "fonts/roboto-cyrillic-wght-normal.woff2": "font/woff2",
	} {
		w := httptest.NewRecorder()
		handled := app.handleSubscriptionRequest(w, httptest.NewRequest("GET", "/sub/_assets/"+path, nil))
		if !handled || w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), typ) || w.Body.Len() == 0 {
			t.Fatalf("%s handled=%v status=%d", path, handled, w.Code)
		}
	}
	for _, path := range []string{"fonts/LICENSE-Geist.txt", "fonts/../theme-md3.css", "fonts/x.woff2/../../main.go", "theme-md3.css", "icons-lucide.js", "index.html", "../index.html", "../../state.json"} {
		w := httptest.NewRecorder()
		app.handleSubscriptionRequest(w, httptest.NewRequest("GET", "/sub/_assets/"+path, nil))
		if w.Code == 200 && w.Body.Len() > 0 {
			t.Fatalf("private path exposed: %s", path)
		}
	}
}

func TestThemeIconSetsCoverEverySymbol(t *testing.T) {
	symbolID := regexp.MustCompile(`<symbol id=\\"(sf-[a-z0-9-]+)\\"`)
	ids := func(name string) map[string]bool {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		for _, m := range symbolID.FindAllStringSubmatch(string(raw), -1) {
			found[m[1]] = true
		}
		return found
	}
	base := ids("web/sf-symbols.js")
	if len(base) < 50 {
		t.Fatalf("only %d icons", len(base))
	}
	for _, set := range []string{"web/icons-lucide.js", "web/icons-material.js"} {
		themed := ids(set)
		for id := range base {
			if !themed[id] {
				t.Fatalf("%s missing %s", set, id)
			}
		}
	}
}

func TestThemeSegmentedRulesCoverAddedSecurityButtons(t *testing.T) {
	raw, err := os.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "extras.id='security-extra-toggles'") {
		t.Fatal("extra security entry disappeared")
	}
	css, err := os.ReadFile("web/components.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".segmented, .liquid-tabbar, #security-extra-toggles) > button {", ".segmented, .liquid-tabbar, #security-extra-toggles) > button.active {"} {
		if !strings.Contains(string(css), want) {
			t.Fatalf("extra security controls lost state rule %s", want)
		}
	}
}
