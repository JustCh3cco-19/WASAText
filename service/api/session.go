package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/JustCh3cco-19/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		Name     string `json:"name"`
		Password string `json:"password"`
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
	if len(payload.Password) < 6 || len(payload.Password) > 128 {
		writeError(w, http.StatusBadRequest, "Password must be between 6 and 128 characters")
		return
	}

	user, token, err := rt.db.LoginUser(r.Context(), payload.Name, payload.Password)
	if err != nil {
		if errors.Is(err, database.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		rt.respondDBError(w, ctx, err, "")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user":  toUserResponse(user),
	})
}
