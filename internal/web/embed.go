package web

import (
	"embed"
	"io/fs"
	"net/http"
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
// with SPA routing support (serves index.html for all non-file routes)
func SPAHandler() http.Handler {
	fileServer := http.FileServer(GetFileSystem())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file
		path := r.URL.Path

		// Check if file exists
		f, err := GetFileSystem().Open(path)
		if err != nil {
			// File doesn't exist, serve index.html for SPA routing
			r.URL.Path = "/"
		} else {
			f.Close()
		}

		fileServer.ServeHTTP(w, r)
	})
}
