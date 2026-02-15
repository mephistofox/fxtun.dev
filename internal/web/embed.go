package web

import (
	"embed"
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

		// Fall back to SPA shell (ru.html for fxtun.ru, index.html for others)
		if ruDomain {
			r.URL.Path = "/ru.html"
		} else {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
