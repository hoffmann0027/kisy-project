//go:build integration

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Audit A-46: nginx limited `location /api/auth/` to 5 r/s, but every auth
// route lives under /api/v1/auth/ — the brute-force limit applied to nothing,
// and nobody noticed because a location that matches nothing is not an error.
// Each prefix location the proxy configs declare must cover a real route.
func TestEveryNginxLocationMatchesARealRoute(t *testing.T) {
	routes := walkRoutesRaw(t, fullRouter(t))
	location := regexp.MustCompile(`(?m)^\s*location\s+(/[^\s{]*)\s*\{`)

	for _, name := range []string{"nginx.conf", "nginx.tls.conf"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "nginx", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range location.FindAllStringSubmatch(string(raw), -1) {
			prefix := m[1]
			if prefix == "/" {
				continue // the SPA, served by the frontend container
			}
			matched := false
			for _, rt := range routes {
				if strings.HasPrefix(rt.raw, prefix) {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("%s: location %s matches no route the backend serves", name, prefix)
			}
		}
	}
}
