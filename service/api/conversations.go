package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleListConversations(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	convs, err := rt.db.ListConversations(r.Context(), ctx.AuthenticatedUser.ID)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	response := struct {
		Conversations []conversationSummaryResponse `json:"conversations"`
	}{
		Conversations: make([]conversationSummaryResponse, 0, len(convs)),
	}
	for _, c := range convs {
		response.Conversations = append(response.Conversations, toConversationSummaryResponse(c))
	}
	writeJSON(w, http.StatusOK, response)
}

func (rt *_router) handleStartConversation(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		SenderID    string `json:"senderId"`
		RecipientID string `json:"recipientId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	payload.SenderID = strings.TrimSpace(payload.SenderID)
	payload.RecipientID = strings.TrimSpace(payload.RecipientID)
	if payload.SenderID == "" || payload.RecipientID == "" {
		writeError(w, http.StatusBadRequest, "Sender and recipient are required")
		return
	}
	if payload.SenderID != ctx.AuthenticatedUser.ID {
		writeError(w, http.StatusForbidden, "Sender does not match authenticated user")
		return
	}

	details, err := rt.db.EnsureDirectConversation(r.Context(), payload.SenderID, payload.RecipientID)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusCreated, toConversationDetailsResponse(details))
}

func (rt *_router) handleGetConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := strings.TrimSpace(ps.ByName("conversationId"))
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "Conversation ID is required")
		return
	}
	details, err := rt.db.GetConversationDetails(r.Context(), ctx.AuthenticatedUser.ID, conversationID)
	if err != nil {
		rt.respondDBError(w, ctx, err, "Conversation not found")
		return
	}
	writeJSON(w, http.StatusOK, toConversationDetailsResponse(details))
}
