package api

import (
	"fmt"
	"html"
	"strings"
	"unicode"
	"unicode/utf8"
)

// stationMonogram renders a compact, brand-consistent fallback icon for
// stations whose real favicon cannot be fetched. It keeps every station
// image-bearing instead of exposing a bare initial in the UI.
func stationMonogram(name, domain string) []byte {
	initial := "?"
	pick := func(v string) (string, bool) {
		for _, r := range strings.TrimSpace(v) {
			if unicode.IsLetter(r) {
				return string(unicode.ToUpper(r)), true
			}
		}
		return "", false
	}
	if c, ok := pick(name); ok {
		initial = c
	} else if c, ok := pick(domain); ok {
		initial = c
	} else if name != "" {
		r, _ := utf8.DecodeRuneInString(name)
		initial = string(r)
	}
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64">
  <rect width="64" height="64" rx="15" fill="#0a1014"/>
  <circle cx="32" cy="32" r="25" fill="none" stroke="#2ee6a6" stroke-opacity=".28"/>
  <circle cx="32" cy="32" r="31" fill="none" stroke="#2ee6a6" stroke-opacity=".10"/>
  <text x="32" y="41" text-anchor="middle" font-family="Inter, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif" font-size="26" font-weight="700" fill="#2ee6a6">%s</text>
</svg>`, html.EscapeString(initial)))
}
