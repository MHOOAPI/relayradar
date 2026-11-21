package favicon

import "testing"

func TestParseIconHref(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string
	}{
		{
			name: "apple touch icon wins",
			html: `<html><head>
				<link rel="icon" href="/favicon-32.png">
				<link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png">
			</head></html>`,
			want: "https://example.com/apple-touch-icon.png",
		},
		{
			name: "relative icon resolved",
			html: `<head><link rel="shortcut icon" href="assets/icon.svg"></head>`,
			want: "https://example.com/assets/icon.svg",
		},
		{
			name: "absolute cdn icon",
			html: `<link href="https://cdn.example.net/i.png" rel="icon">`,
			want: "https://cdn.example.net/i.png",
		},
		{
			name: "data uri ignored",
			html: `<link rel="icon" href="data:image/png;base64,AAAA">`,
			want: "",
		},
		{
			name: "no icon",
			html: `<link rel="stylesheet" href="/app.css">`,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseIconHref([]byte(tc.html), "https://example.com/")
			if got != tc.want {
				t.Fatalf("parseIconHref = %q, want %q", got, tc.want)
			}
		})
	}
}
