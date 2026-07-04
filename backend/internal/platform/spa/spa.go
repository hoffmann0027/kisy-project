// Package spa serves a built single-page application from a directory,
// falling back to index.html for client-side routes. This lets the backend
// host the frontend on the same origin — required for a single-service
// deploy where SameSite=Strict cookies must be first-party.
package spa

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Handler serves static assets from dir and returns index.html for any
// path that does not map to an existing file (SPA history routing). Assets
// under /assets get long-lived immutable caching; index.html is never
// cached so new deploys are picked up immediately.
func Handler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	indexPath := filepath.Join(dir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)
		full := filepath.Join(dir, clean)

		// Guard against path traversal outside the web root.
		if !strings.HasPrefix(full, filepath.Clean(dir)) {
			http.NotFound(w, r)
			return
		}

		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, indexPath)
	})
}
