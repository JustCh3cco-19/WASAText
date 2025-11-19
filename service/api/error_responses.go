package api

import (
	"errors"
	"net/http"

	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/api/reqcontext"
	"git.sapienzaapps.it/fantasticcoffee/fantastic-coffee-decaffeinated/service/database"
)

func (rt *_router) respondDBError(w http.ResponseWriter, ctx reqcontext.RequestContext, err error, notFoundMessage string) {
	switch {
	case errors.Is(err, database.ErrNotFound):
		if notFoundMessage == "" {
			notFoundMessage = "Resource not found"
		}
		writeError(w, http.StatusNotFound, notFoundMessage)
	case errors.Is(err, database.ErrForbidden):
		writeError(w, http.StatusForbidden, "Forbidden")
	case errors.Is(err, database.ErrConflict):
		writeError(w, http.StatusConflict, "Conflict")
	case errors.Is(err, database.ErrBadRequest):
		writeError(w, http.StatusBadRequest, "Invalid request data")
	default:
		ctx.Logger.WithError(err).Error("internal server error")
		writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}
