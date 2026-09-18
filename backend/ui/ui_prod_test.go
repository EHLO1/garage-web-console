//go:build prod

package ui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Run after copying the frontend dist into ui/dist, as in the Docker build.
func TestSPAAtRuntimeBasePaths(t *testing.T) {
	for _, base := range []string{"", "/console", "/tools/garage"} {
		t.Run("base="+base, func(t *testing.T) {
			t.Setenv("BASE_PATH", base)
			mux := http.NewServeMux()
			ServeUI(mux)
			for _, route := range []string{"/", "/index.html", "/auth/login", "/buckets/test?prefix=photos%2F"} {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest("GET", base+route, nil))
				body := rec.Body.String()
				if rec.Code != http.StatusOK || !strings.Contains(body, "<!doctype html>") {
					t.Fatalf("%s: expected SPA HTML, got %d", route, rec.Code)
				}
				if strings.Contains(body, "/__garage_base__") || !strings.Contains(body, base+"/_app/immutable/") {
					t.Fatalf("%s: incorrect asset base", route)
				}
			}
			assets := 0
			err := fs.WalkDir(embeddedFs, "dist/_app", func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || !(strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css")) {
					return nil
				}
				assets++
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest("GET", base+strings.TrimPrefix(path, "dist"), nil))
				if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "/__garage_base__") || strings.HasPrefix(strings.TrimSpace(rec.Body.String()), "<!doctype html>") {
					t.Errorf("asset %s not embedded or rewritten correctly", path)
				}
				return nil
			})
			if err != nil || assets == 0 {
				t.Fatalf("SvelteKit _app assets missing: %v", err)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest("GET", base+"/_app/missing.js", nil))
			if rec.Code != http.StatusNotFound {
				t.Fatal("missing assets must return 404")
			}
		})
	}
}
