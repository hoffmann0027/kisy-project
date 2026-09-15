// Package spa serves a built single-page application from a directory,
// falling back to index.html for client-side routes. This lets the backend
// host the frontend on the same origin — required for a single-service
// deploy where SameSite=Strict cookies must be first-party.
package spa

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// resolve maps a request path to a file inside dir, and can only ever name a
// file inside it.
//
// The containment is structural rather than a check after the fact: a URL path
// is rooted at "/" before it is cleaned, so every ".." is resolved against that
// root and the leftovers are dropped — "/a/../../../etc/passwd" becomes
// "/etc/passwd", which then joins onto dir. It is the same approach net/http
// uses for its own file server, and it does not depend on the caller having
// normalised anything: chi hands over the raw (percent-decoded) path, where a
// traversal attempt arrives intact.
func resolve(dir, urlPath string) string {
	return filepath.Join(dir, filepath.FromSlash(path.Clean("/"+urlPath)))
}

// Handler serves static assets from dir and returns index.html for any
// path that does not map to an existing file (SPA history routing). Assets
// under /assets get long-lived immutable caching; index.html is never
// cached so new deploys are picked up immediately.
func Handler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	indexPath := filepath.Join(dir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		full := resolve(dir, r.URL.Path)

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
