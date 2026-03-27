package coordinator

import (
	"io/fs"
	"net/http"
	"strings"

	claw_mesh "github.com/SallyKAN/claw-mesh"
)

// DashboardHandler returns an http.Handler that serves the embedded web dashboard.
// For the root index.html it injects the coordinator token so the SPA can
// authenticate against mutating API endpoints.
func DashboardHandler(token string) http.Handler {
	sub, err := fs.Sub(claw_mesh.WebDist, "web/dist")
	if err != nil {
		panic("failed to load embedded dashboard: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	// Pre-inject token into all page index.html files (SPA sub-routes).
	injectedPages := make(map[string][]byte)
	fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == "index.html" {
			raw, readErr := fs.ReadFile(sub, path)
			if readErr != nil {
				return nil
			}
			injected := strings.Replace(
				string(raw),
				"<head>",
				"<head><script>window.__TOKEN__=\""+token+"\";</script>",
				1,
			)
			// Normalize: "chat/index.html" -> "/chat/"
			servePath := "/" + strings.TrimSuffix(path, "index.html")
			injectedPages[servePath] = []byte(injected)
			injectedPages[servePath+"index.html"] = []byte(injected)
		}
		return nil
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if page, ok := injectedPages[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(page)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
