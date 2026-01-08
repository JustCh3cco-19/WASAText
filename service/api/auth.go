package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/JustCh3cco-19/WASAText/service/database"
	"github.com/julienschmidt/httprouter"
)

type authedHandler func(http.ResponseWriter, *http.Request, httprouter.Params, reqcontext.RequestContext)

var (
	errMissingToken = errors.New("missing bearer token")
	errInvalidToken = errors.New("invalid bearer token")
)

func (rt *_router) wrapAuthenticated(fn authedHandler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return rt.wrap(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
		user, err := rt.authenticate(r)
		if err != nil {
			rt.handleAuthError(w, ctx, err)
			return
		}
		ctx.AuthenticatedUser = &reqcontext.AuthenticatedUser{
			ID:    user.ID,
			Name:  user.Name,
			Photo: user.Photo,
		}
		fn(w, r, ps, ctx)
	})
}

func (rt *_router) authenticate(r *http.Request) (database.User, error) {
	header := r.Header.Get("Authorization")
	if strings.TrimSpace(header) == "" {
		return database.User{}, errMissingToken
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return database.User{}, errInvalidToken
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return database.User{}, errInvalidToken
	}
	user, err := rt.db.GetUserByToken(r.Context(), token)
	if err != nil {
		return database.User{}, err
	}
	return user, nil
}

func (rt *_router) handleAuthError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error) {
	switch {
	case errors.Is(err, errMissingToken):
		writeError(w, http.StatusUnauthorized, "Token di autenticazione mancante")
	case errors.Is(err, errInvalidToken):
		writeError(w, http.StatusUnauthorized, "Token di autenticazione non valido")
	case errors.Is(err, database.ErrNotFound):
		writeError(w, http.StatusUnauthorized, "Token di autenticazione non valido o mancante")
	case errors.Is(err, database.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "Token di autenticazione non valido o mancante")
	default:
		ctx.Logger.WithError(err).Error("authentication error")
		writeError(w, http.StatusInternalServerError, "Errore interno del server")
	}
}
