package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleCommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := strings.TrimSpace(ps.ByName("conversationId"))
	messageID := strings.TrimSpace(ps.ByName("messageId"))
	if conversationID == "" || messageID == "" {
		writeError(w, http.StatusBadRequest, "Conversation ID and message ID are required")
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	payload.Content = strings.TrimSpace(payload.Content)
	if payload.Content == "" || len(payload.Content) > 500 {
		writeError(w, http.StatusBadRequest, "Content must be between 1 and 500 characters")
		return
	}
	if err := rt.db.AddComment(r.Context(), conversationID, messageID, ctx.AuthenticatedUser.ID, payload.Content); err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *_router) handleUncommentMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := strings.TrimSpace(ps.ByName("conversationId"))
	messageID := strings.TrimSpace(ps.ByName("messageId"))
	if conversationID == "" || messageID == "" {
		writeError(w, http.StatusBadRequest, "Conversation ID and message ID are required")
		return
	}
	if err := rt.db.RemoveComment(r.Context(), conversationID, messageID, ctx.AuthenticatedUser.ID); err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
