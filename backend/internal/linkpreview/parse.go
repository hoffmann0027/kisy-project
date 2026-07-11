package linkpreview

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

// Preview is the metadata returned for a URL.
type Preview struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	SiteName    string `json:"siteName"`
}

// A <meta> tag with property/name and content in either attribute order.
// Go's regexp is RE2 (linear time, no catastrophic backtracking) and the
// input is already capped at maxHTMLBytes, so unbounded quantifiers are safe.
var (
	metaTag   = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	attrProp  = regexp.MustCompile(`(?is)\b(?:property|name)\s*=\s*["']([^"']*)["']`)
	attrValue = regexp.MustCompile(`(?is)\bcontent\s*=\s*["']([^"']*)["']`)
	titleTag  = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	headEnd   = regexp.MustCompile(`(?is)</head>`)
)

// parse extracts OpenGraph/Twitter/standard metadata from HTML. Only the
// <head> is scanned (bounded); values are HTML-unescaped and trimmed.
// imageURL is resolved to an absolute URL against the page's final URL.
func parse(htmlBytes []byte, finalURL *url.URL) Preview {
	doc := string(htmlBytes)
	if loc := headEnd.FindStringIndex(doc); loc != nil {
		doc = doc[:loc[1]]
	}

	meta := map[string]string{}
	for _, tag := range metaTag.FindAllString(doc, -1) {
		p := attrProp.FindStringSubmatch(tag)
		v := attrValue.FindStringSubmatch(tag)
		if p == nil || v == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(p[1]))
		if _, seen := meta[key]; !seen {
			meta[key] = clean(v[1])
		}
	}

	pick := func(keys ...string) string {
		for _, k := range keys {
			if val := meta[k]; val != "" {
				return val
			}
		}
		return ""
	}

	title := pick("og:title", "twitter:title")
	if title == "" {
		if m := titleTag.FindStringSubmatch(doc); m != nil {
			title = clean(m[1])
		}
	}

	p := Preview{
		URL:         finalURL.String(),
		Title:       title,
		Description: pick("og:description", "twitter:description", "description"),
		SiteName:    pick("og:site_name", "application-name"),
		ImageURL:    absolute(finalURL, pick("og:image", "og:image:url", "twitter:image", "twitter:image:src")),
	}
	// Fall back to the host as the site name so a bare card still identifies
	// its origin.
	if p.SiteName == "" {
		p.SiteName = finalURL.Hostname()
	}
	return p
}

func clean(s string) string {
	return strings.TrimSpace(html.UnescapeString(s))
}

// absolute resolves a possibly-relative image URL against the page URL and
// drops anything that is not http(s) (e.g. data: URIs are ignored — the
// image proxy only fetches remote http(s) images).
func absolute(base *url.URL, ref string) string {
	if ref == "" {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(u)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}
