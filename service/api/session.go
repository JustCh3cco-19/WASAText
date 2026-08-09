package api

import (
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		return
	}

	payload.Name = strings.TrimSpace(payload.Name)
	if len(payload.Name) < 3 || len(payload.Name) > 16 || !usernameRegexp.MatchString(payload.Name) {
		writeError(w, http.StatusBadRequest, "Il nome deve essere tra 3 e 16 caratteri e contenere solo lettere, numeri o underscore")
		return
	}

	if len(payload.Password) < 10 || len(payload.Password) > 128 {
		writeError(w, http.StatusBadRequest, "La password deve contenere tra 10 e 128 caratteri")
		return
	}
	user, token, err := rt.db.LoginUser(r.Context(), payload.Name, payload.Password)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}

	setSessionCookie(w, r, token)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user": toUserResponse(user),
	})
}

func (rt *_router) handleRegister(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		return
	}
	payload.Name = strings.TrimSpace(payload.Name)
	if len(payload.Name) < 3 || len(payload.Name) > 16 || !usernameRegexp.MatchString(payload.Name) {
		writeError(w, http.StatusBadRequest, "Nome utente non valido")
		return
	}
	if len(payload.Password) < 10 || len(payload.Password) > 128 {
		writeError(w, http.StatusBadRequest, "La password deve contenere tra 10 e 128 caratteri")
		return
	}
	user, token, err := rt.db.RegisterUser(r.Context(), payload.Name, payload.Password)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	setSessionCookie(w, r, token)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"user": toUserResponse(user)})
}

func (rt *_router) handleLogout(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	token, err := bearerToken(r)
	if err != nil {
		rt.handleAuthError(w, ctx, err)
		return
	}
	if err := rt.db.RevokeToken(r.Context(), token); err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode, MaxAge: sessionMaxAge,
	})
}
