// Package filehttp hands user-uploaded bytes back to a browser that must
// never run them.
//
// Message attachments, note files and post media come from the API origin —
// the same origin as the app in a browser. A file served as text/html there,
// the moment any one of its other defences slips (a proxy dropping nosniff, a
// WebView ignoring Content-Disposition), is stored XSS on that origin (audit
// A-41). So every such response is built here, one way.
package filehttp

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// sandboxCSP makes the file, should a browser ever render it as a page, a
// document in an opaque origin with nothing to run and nothing to load.
const sandboxCSP = "sandbox; default-src 'none'; img-src 'self' data:; media-src 'self'; style-src 'unsafe-inline'"

// ServeType is the Content-Type a stored file goes out with: pictures, audio
// and video keep their sniffed type and render in place; anything else is an
// opaque download, whatever it is.
func ServeType(mime string) (contentType string, inline bool) {
	for _, media := range []string{"image/", "audio/", "video/"} {
		if strings.HasPrefix(mime, media) && !strings.HasPrefix(mime, "image/svg") {
			return mime, true
		}
	}
	return "application/octet-stream", false
}

// Write sends data as a stored file named name, whose type was sniffed from
// the bytes as mime. name must already be safe for a header value; it is
// percent-encoded here.
func Write(w http.ResponseWriter, mime, name string, data []byte, cacheControl string) {
	contentType, inline := ServeType(mime)
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	h := w.Header()
	h.Set("Content-Type", contentType)
	h.Set("Content-Length", strconv.Itoa(len(data)))
	h.Set("Cache-Control", cacheControl)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", sandboxCSP)
	h.Set("Content-Disposition", disposition+`; filename*=UTF-8''`+url.PathEscape(name))
	w.WriteHeader(http.StatusOK)
	// #nosec G705 -- user-uploaded bytes, never handed to the browser as
	// active content: anything but a picture, audio or video goes out as
	// application/octet-stream and an attachment, with nosniff, and the
	// sandbox CSP strips scripts and origin from whatever does render.
	_, _ = w.Write(data)
}
