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

// SPAHandler returns an http.Handler that serves the embedded web UI
// with SPA routing support. It checks for prerendered HTML files first
// (e.g. /login → login.html), then falls back to index.html for SPA routing.
func SPAHandler() http.Handler {
	fileServer := http.FileServer(GetFileSystem())
	filesystem := GetFileSystem()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try exact file first (static assets like /assets/app.js)
		f, err := filesystem.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try prerendered HTML (e.g. /login → /login.html)
		if path != "/" {
			htmlPath := strings.TrimSuffix(path, "/") + ".html"
			if f, err := filesystem.Open(htmlPath); err == nil {
				f.Close()
				r.URL.Path = htmlPath
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fall back to index.html for SPA routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
