//go:build prod
// +build prod

package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed all:dist
var embeddedFs embed.FS

func ServeUI(mux *http.ServeMux) {
	distFs, _ := fs.Sub(embeddedFs, "dist")
	fileServer := http.FileServer(http.FS(distFs))
	basePath := os.Getenv("BASE_PATH")

	mux.Handle(basePath+"/", http.StripPrefix(basePath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_path := path.Clean(r.URL.Path)[1:]

		// Serve the SvelteKit SPA fallback for client routes and the root.
		_, statErr := fs.Stat(distFs, _path)
		if statErr != nil && strings.HasPrefix(_path, "_app/") {
			http.NotFound(w, r)
			return
		}
		if _path == "" || _path == "index.html" || statErr != nil {
			index, _ := fs.ReadFile(distFs, "index.html")
			html := string(index)

			html = replaceBasePath(html, basePath)

			w.Header().Add("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
			return
		}

		// SvelteKit embeds its build-time base in generated JS and CSS.
		// Substitute the marker even when serving from the origin root.
		if strings.HasSuffix(_path, ".js") || strings.HasSuffix(_path, ".css") {
			data, _ := fs.ReadFile(distFs, _path)
			html := string(data)
			html = replaceBasePath(html, basePath)

			w.Header().Add("Content-Type", "text/javascript")
			if strings.HasSuffix(_path, ".css") {
				w.Header().Set("Content-Type", "text/css")
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(html))
			return
		}

		fileServer.ServeHTTP(w, r)
	})))
}

func replaceBasePath(content string, basePath string) string {
	return strings.ReplaceAll(content, "/__garage_base__", basePath)
}
