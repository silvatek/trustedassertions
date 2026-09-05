package web

import (
	"net/http"
	"os"
	"strings"
)

const defaultRevision = "dev"

type healthResponse struct {
	Status   string `json:"status"`
	Revision string `json:"revision"`
}

func HealthWebHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	writeJSON(w, http.StatusOK, healthResponse{
		Status:   "ok",
		Revision: commitRevision(),
	})
}

func commitRevision() string {
	if sha := strings.TrimSpace(os.Getenv("COMMIT_SHA")); sha != "" {
		return sha
	}
	return defaultRevision
}
