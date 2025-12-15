package api

import (
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleSearchUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	query := strings.TrimSpace(r.URL.Query().Get("username"))
	exclude := ""
	if ctx.AuthenticatedUser != nil {
		exclude = ctx.AuthenticatedUser.ID
	}
	users, err := rt.db.SearchUsers(r.Context(), query, exclude)
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
