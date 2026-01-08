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
		writeError(w, http.StatusBadRequest, "Payload JSON non valido")
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if len(payload.Name) < 3 || len(payload.Name) > 16 || !usernameRegexp.MatchString(payload.Name) {
		writeError(w, http.StatusBadRequest, "Il nome deve essere tra 3 e 16 caratteri e contenere solo lettere, numeri o underscore")
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
