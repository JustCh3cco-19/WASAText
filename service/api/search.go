package api

import (
	"net/http"
	"strings"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleSearchUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	query := strings.TrimSpace(r.URL.Query().Get("username"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "username query parameter is required")
		return
	}
	users, err := rt.db.SearchUsers(r.Context(), query)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	response := struct {
		Users []userResponse `json:"users"`
	}{
		Users: make([]userResponse, 0, len(users)),
	}
	for _, u := range users {
		response.Users = append(response.Users, toUserResponse(u))
	}
	writeJSON(w, http.StatusOK, response)
}
