package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/JustCh3cco-19/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
)

var usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (rt *_router) handleUpdateUserPhoto(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.AuthenticatedUser == nil {
		writeError(w, http.StatusUnauthorized, "Token di autenticazione mancante")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBinaryPayload))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Impossibile leggere il payload della foto")
		return
	}
	photo := strings.TrimSpace(string(body))
	if photo == "" {
		writeError(w, http.StatusBadRequest, "Il payload della foto non puo essere vuoto")
		return
	}

	user, err := rt.db.UpdateUserPhoto(r.Context(), ctx.AuthenticatedUser.ID, photo)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (rt *_router) handleUpdateUserName(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.AuthenticatedUser == nil {
		writeError(w, http.StatusUnauthorized, "Token di autenticazione mancante")
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Payload JSON non valido")
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	if len(payload.Name) < 3 || len(payload.Name) > 16 || !usernameRegexp.MatchString(payload.Name) {
		writeError(w, http.StatusBadRequest, "Nome utente non valido")
		return
	}

	user, err := rt.db.UpdateUserName(r.Context(), ctx.AuthenticatedUser.ID, payload.Name)
	if err != nil {
		if errors.Is(err, database.ErrConflict) {
			writeError(w, http.StatusConflict, "Nome utente gia in uso")
			return
		}
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}
