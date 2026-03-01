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

	// Read index.html once at startup for token injection.
	indexBytes, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic("failed to read embedded index.html: " + err.Error())
	}
	indexHTML := strings.Replace(
		string(indexBytes),
		"<head>",
		"<head><script>window.__TOKEN__=\""+token+"\";</script>",
		1,
	)

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve injected index.html for root or index.html requests.
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(indexHTML))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
