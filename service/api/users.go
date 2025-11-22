package api

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

var usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (rt *_router) handleUpdateUserPhoto(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	if ctx.AuthenticatedUser == nil {
		writeError(w, http.StatusUnauthorized, "Missing authentication token")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBinaryPayload))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Unable to read photo payload")
		return
	}
	photo := strings.TrimSpace(string(body))
	if photo == "" {
		writeError(w, http.StatusBadRequest, "Photo payload cannot be empty")
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
		writeError(w, http.StatusUnauthorized, "Missing authentication token")
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	if len(payload.Name) < 3 || len(payload.Name) > 16 || !usernameRegexp.MatchString(payload.Name) {
		writeError(w, http.StatusBadRequest, "Invalid username")
		return
	}

	user, err := rt.db.UpdateUserName(r.Context(), ctx.AuthenticatedUser.ID, payload.Name)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}
