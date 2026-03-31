package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// GetFileSystem returns the embedded file system for the web UI
func GetFileSystem() http.FileSystem {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}

// Handler returns an http.Handler that serves the embedded web UI
func Handler() http.Handler {
	return http.FileServer(GetFileSystem())
}

// isRuDomain checks if the request host is fxtun.ru or a subdomain of it.
func isRuDomain(host string) bool {
	h, _, _ := strings.Cut(host, ":")
	return h == "fxtun.ru" || strings.HasSuffix(h, ".fxtun.ru")
}

// SPAHandler returns an http.Handler that serves the embedded web UI
// with SPA routing support and domain-based pre-rendering.
//
// For fxtun.ru requests, it serves Russian prerendered pages (ru.html, ru/login.html, etc.)
// instead of the default English versions, so that crawlers see the correct language and
// canonical URLs without relying on client-side JavaScript.
//
// Routing priority:
//  1. Domain-based root: fxtun.ru/ → ru.html
//  2. Exact file match: /assets/app.js → assets/app.js
//  3. Domain-based prerendered: fxtun.ru/login → ru/login.html
//  4. Default prerendered: /login → login.html
//  5. SPA fallback: fxtun.ru → ru.html, fxtun.dev → index.html
func SPAHandler() http.Handler {
	fileServer := http.FileServer(GetFileSystem())
	filesystem := GetFileSystem()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		ruDomain := isRuDomain(r.Host)

		// Normalize trailing slash: /pricing/ → 301 → /pricing
		// Exclude root "/" and /blog/ (served by nginx with trailing slash convention)
		if path != "/" && strings.HasSuffix(path, "/") && !strings.HasPrefix(path, "/blog") {
			http.Redirect(w, r, strings.TrimRight(path, "/"), http.StatusMovedPermanently)
			return
		}

		// Redirect redundant locale prefixes:
		// fxtun.ru/ru/* → fxtun.ru/* (already Russian)
		// fxtun.dev/en/* → fxtun.dev/* (already English)
		if (ruDomain && strings.HasPrefix(path, "/ru")) || (!ruDomain && strings.HasPrefix(path, "/en")) {
			prefix := "/ru"
			if !ruDomain {
				prefix = "/en"
			}
			newPath := strings.TrimPrefix(path, prefix)
			if newPath == "" {
				newPath = "/"
			}
			http.Redirect(w, r, newPath, http.StatusMovedPermanently)
			return
		}

		// Cross-domain locale redirects:
		// fxtun.ru/en/* → fxtun.dev/* (wrong language for this domain)
		// fxtun.dev/ru/* → fxtun.ru/* (wrong language for this domain)
		if ruDomain && strings.HasPrefix(path, "/en") {
			newPath := strings.TrimPrefix(path, "/en")
			if newPath == "" {
				newPath = "/"
			}
			http.Redirect(w, r, "https://fxtun.dev"+newPath, http.StatusMovedPermanently)
			return
		}
		if !ruDomain && strings.HasPrefix(path, "/ru") {
			newPath := strings.TrimPrefix(path, "/ru")
			if newPath == "" {
				newPath = "/"
			}
			http.Redirect(w, r, "https://fxtun.ru"+newPath, http.StatusMovedPermanently)
			return
		}

		// Serve domain-aware sitemap.xml with hreflang links
		if path == "/sitemap.xml" {
			serveDomainSitemap(w, filesystem, ruDomain)
			return
		}

		// Serve domain-aware robots.txt: keep only sitemaps for the current domain
		if path == "/robots.txt" {
			serveDomainRobots(w, r, filesystem)
			return
		}

		// Serve Russian llms.txt for fxtun.ru
		if path == "/llms.txt" && ruDomain {
			r.URL.Path = "/llms-ru.txt"
			fileServer.ServeHTTP(w, r)
			return
		}
		if path == "/llms-full.txt" && ruDomain {
			r.URL.Path = "/llms-full-ru.txt"
			fileServer.ServeHTTP(w, r)
			return
		}

		// Domain-based root: serve ru.html for fxtun.ru/
		if path == "/" && ruDomain {
			if f, err := filesystem.Open("/ru.html"); err == nil {
				f.Close()
				r.URL.Path = "/ru.html"
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Try exact file first (static assets like /assets/app.js)
		f, err := filesystem.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try prerendered HTML
		if path != "/" {
			// For fxtun.ru, try RU prerendered version first (e.g. /login → /ru/login.html)
			if ruDomain {
				ruHTMLPath := "/ru" + strings.TrimSuffix(path, "/") + ".html"
				if f, err := filesystem.Open(ruHTMLPath); err == nil {
					f.Close()
					r.URL.Path = ruHTMLPath
					fileServer.ServeHTTP(w, r)
					return
				}
			}

			// Try default prerendered HTML (e.g. /login → /login.html)
			htmlPath := strings.TrimSuffix(path, "/") + ".html"
			if f, err := filesystem.Open(htmlPath); err == nil {
				f.Close()
				r.URL.Path = htmlPath
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Only serve SPA shell for known client-side routes; return 404 for unknown paths
		if !isKnownSPARoute(path) {
			http.NotFound(w, r)
			return
		}

		if ruDomain {
			r.URL.Path = "/ru.html"
		} else {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// spaRoutePrefixes lists paths that are handled by the Vue SPA router
// and require the SPA shell (not pre-rendered by SSG).
var spaRoutePrefixes = []string{
	"/checkout",
	"/dashboard",
	"/inspect/",
	"/domains",
	"/tokens",
	"/downloads/files",
	"/profile",
	"/auth/",
	"/admin/",
}

// isKnownSPARoute checks if the path matches a known client-side route
// that should receive the SPA shell instead of a 404.
func isKnownSPARoute(path string) bool {
	for _, prefix := range spaRoutePrefixes {
		if path == prefix || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// serveDomainSitemap reads sitemap.xml from the embedded FS, injects
// hreflang xhtml:link alternate entries for ru/en/x-default, and for
// fxtun.ru replaces <loc> URLs from fxtun.dev to fxtun.ru.
func serveDomainSitemap(w http.ResponseWriter, filesystem http.FileSystem, ruDomain bool) {
	f, err := filesystem.Open("/sitemap.xml")
	if err != nil {
		http.Error(w, "sitemap not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "failed to read sitemap", http.StatusInternalServerError)
		return
	}

	content := string(data)

	// Inject hreflang links: for each <loc>https://fxtun.dev/path</loc>,
	// add xhtml:link alternates before </url>.
	var result strings.Builder
	result.Grow(len(content) * 2)

	remaining := content
	for {
		urlEnd := strings.Index(remaining, "</url>")
		if urlEnd == -1 {
			result.WriteString(remaining)
			break
		}

		// Extract the chunk up to </url>
		chunk := remaining[:urlEnd]

		// Find <loc>...</loc> within this chunk
		locStart := strings.Index(chunk, "<loc>")
		locEnd := strings.Index(chunk, "</loc>")
		if locStart != -1 && locEnd != -1 {
			loc := chunk[locStart+5 : locEnd]
			// Extract path from URL (e.g. https://fxtun.dev/pricing → /pricing)
			path := strings.TrimPrefix(loc, "https://fxtun.dev")
			if path == "" {
				path = "/"
			}

			enURL := "https://fxtun.dev" + path
			ruURL := "https://fxtun.ru" + path

			// For fxtun.ru, replace the <loc> URL
			if ruDomain {
				chunk = strings.Replace(chunk, loc, ruURL, 1)
			}

			hreflang := `<xhtml:link rel="alternate" hreflang="en" href="` + enURL + `"/>` +
				`<xhtml:link rel="alternate" hreflang="ru" href="` + ruURL + `"/>` +
				`<xhtml:link rel="alternate" hreflang="x-default" href="` + enURL + `"/>`

			result.WriteString(chunk)
			result.WriteString(hreflang)
		} else {
			result.WriteString(chunk)
		}

		result.WriteString("</url>")
		remaining = remaining[urlEnd+6:]
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(result.String()))
}

// serveDomainRobots reads robots.txt from the embedded FS and filters
// sitemap references to only include the current domain's sitemaps.
func serveDomainRobots(w http.ResponseWriter, r *http.Request, filesystem http.FileSystem) {
	f, err := filesystem.Open("/robots.txt")
	if err != nil {
		http.Error(w, "robots.txt not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "failed to read robots.txt", http.StatusInternalServerError)
		return
	}

	content := string(data)
	if isRuDomain(r.Host) {
		content = strings.ReplaceAll(content, "Sitemap: https://fxtun.dev/sitemap.xml\n", "")
		content = strings.ReplaceAll(content, "Sitemap: https://fxtun.dev/blog/sitemap.xml\n", "")
	} else {
		content = strings.ReplaceAll(content, "Sitemap: https://fxtun.ru/sitemap.xml\n", "")
		content = strings.ReplaceAll(content, "Sitemap: https://fxtun.ru/blog/sitemap.xml\n", "")
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}
