package api

import (
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleListConversations(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	limit, offset, err := parsePage(r, 50, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid pagination")
		return
	}
	convs, err := rt.db.ListConversations(r.Context(), ctx.AuthenticatedUser.ID, limit, offset)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	response := struct {
		Conversations []conversationSummaryResponse `json:"conversations"`
		Page          pageResponse                  `json:"page"`
	}{
		Conversations: make([]conversationSummaryResponse, 0, len(convs)),
		Page:          pageResponse{Limit: limit, Offset: offset},
	}
	for _, c := range convs {
		response.Conversations = append(response.Conversations, toConversationSummaryResponse(c))
	}
	writeJSON(w, http.StatusOK, response)
}

func (rt *_router) handleStartConversation(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var payload struct {
		RecipientID string `json:"recipientId"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		return
	}
	payload.RecipientID = strings.TrimSpace(payload.RecipientID)
	if payload.RecipientID == "" {
		writeError(w, http.StatusBadRequest, "Recipient is required")
		return
	}

	details, err := rt.db.EnsureDirectConversation(r.Context(), ctx.AuthenticatedUser.ID, payload.RecipientID)
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
	limit, offset, err := parsePage(r, 100, 200)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid pagination")
		return
	}
	details, err := rt.db.GetConversationDetails(r.Context(), ctx.AuthenticatedUser.ID, conversationID, limit, offset)
	if err != nil {
		rt.respondDBError(w, ctx, err, "Conversation not found")
		return
	}
	response := toConversationDetailsResponse(details)
	response.Page = pageResponse{Limit: limit, Offset: offset}
	writeJSON(w, http.StatusOK, response)
}
