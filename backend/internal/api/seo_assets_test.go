package api

import "testing"

// Model ids contain dots ("openai--gpt-5.4"), so the SPA handler must not treat
// every dotted path as a missing asset.
func TestIsAssetPath(t *testing.T) {
	cases := map[string]bool{
		"/assets/index-Cp4h6esY.js":          true,
		"/assets/index-Duj2iNfp.css":         true,
		"/favicon.svg":                       true,
		"/share/station-aihubmix.png":        true,
		"/og.png":                            true,
		"/models/openai--gpt-5.4":            false,
		"/models/anthropic--claude-opus-4.1": false,
		"/models/gemini-3.1-flash-image":     false,
		"/models":                            false,
		"/audit/ttft":                        false,
		"/stations/302ai":                    false,
	}
	for path, want := range cases {
		if got := isAssetPath(path); got != want {
			t.Errorf("isAssetPath(%q) = %v, want %v", path, got, want)
		}
	}
}
