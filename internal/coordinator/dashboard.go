package coordinator

import (
	"io/fs"
	"net/http"

	claw_mesh "github.com/SallyKAN/claw-mesh"
)

// DashboardHandler returns an http.Handler that serves the embedded web dashboard.
func DashboardHandler() http.Handler {
	sub, err := fs.Sub(claw_mesh.WebDist, "web/dist")
	if err != nil {
		panic("failed to load embedded dashboard: " + err.Error())
	}
	return http.FileServer(http.FS(sub))
}
