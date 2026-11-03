package app

import (
	"regexp"
	"strings"
)

// The persisted field remains compatible with old backups and API clients.
// All three former skins migrate to T0. Appearance now has one component
// system and five color palettes; light/dark is a browser preference.
var uiThemes = map[string]bool{"t0": true, "t1": true, "t2": true, "t3": true, "t4": true, "shadcn": true, "md3": true, "liquid": true}

func normalizeUITheme(theme string) string {
	theme = strings.TrimSpace(theme)
	if theme == "shadcn" || theme == "md3" || theme == "liquid" {
		return "t0"
	}
	if uiThemes[theme] {
		return theme
	}
	return defaultUITheme
}

// themeOnlyAsset matches a <link> or <script> that belongs to one theme, marked
// data-ui-only="liquid" (or shadcn, md3).
var themeOnlyAsset = regexp.MustCompile(`(?s)<(?:link|script)\b[^>]*\bdata-ui-only="([a-z0-9]+)"[^>]*>(?:\s*</script>)?`)

// themedPage sets the theme of an embedded page and drops the styles and
// scripts of the other themes.
func themedPage(page, theme string) string {
	theme = normalizeUITheme(theme)
	page = strings.Replace(page, `data-color-theme="t0"`, `data-color-theme="`+theme+`"`, 1)
	return themeOnlyAsset.ReplaceAllStringFunc(page, func(tag string) string {
		if themeOnlyAsset.FindStringSubmatch(tag)[1] == theme {
			return tag
		}
		return ""
	})
}

func (a *App) uiTheme() string {
	return normalizeUITheme(a.manager.settingsSnapshot().UITheme)
}

func init() {
	registerServerMessages(map[string][10]string{
		"面板外观主题无效": {"Invalid panel theme", "面板外觀主題無效", "パネルのテーマが無効です", "Неверная тема панели", "پوستهٔ پنل نامعتبر است", "Giao diện bảng điều khiển không hợp lệ", "Tema del panel no válido", "Panel teması geçersiz", "Неправильна тема панелі", "Tema do painel inválido"},
	}, nil)
}
