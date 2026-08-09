package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// liveness is an HTTP handler that checks the API server status. If the server cannot serve requests (e.g., some
// resources are not ready), this should reply with HTTP Status 500. Otherwise, with HTTP Status 200
func (rt *_router) liveness(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) readiness(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	if err := rt.db.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "Database unavailable")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
