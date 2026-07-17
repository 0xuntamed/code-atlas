package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist/*
var assets embed.FS

func Handler() http.Handler {
	root, _ := fs.Sub(assets, "dist")
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.NotFound(w, r)
			return
		}
		requested := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requested == "." || requested == "" {
			requested = "index.html"
		}
		if _, err := fs.Stat(root, requested); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/index.html"
			files.ServeHTTP(w, r2)
			return
		}
		files.ServeHTTP(w, r)
	})
}
