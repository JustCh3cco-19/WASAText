package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if len(payload.Name) < 3 || len(payload.Name) > 16 || !usernameRegexp.MatchString(payload.Name) {
		writeError(w, http.StatusBadRequest, "Name must be between 3 and 16 characters and contain only letters, numbers or underscore")
		return
	}

	user, token, err := rt.db.LoginUser(r.Context(), payload.Name)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user":  toUserResponse(user),
	})
}
