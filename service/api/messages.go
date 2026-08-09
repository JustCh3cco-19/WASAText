package api

import (
	"net/http"
	"strings"

	"github.com/JustCh3cco-19/WASAText/service/api/reqcontext"
	"github.com/JustCh3cco-19/WASAText/service/database"
	"github.com/JustCh3cco-19/WASAText/service/globaltime"
	"github.com/julienschmidt/httprouter"
)

func (rt *_router) handleSendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := strings.TrimSpace(ps.ByName("conversationId"))
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "Conversation ID is required")
		return
	}
	if err := parseMultipart(w, r); err != nil {
		return
	}
	content := strings.TrimSpace(r.FormValue("content"))
	attachment := strings.TrimSpace(r.FormValue("attachment"))
	replyTo := strings.TrimSpace(r.FormValue("replyTo"))
	if content == "" && attachment == "" {
		writeError(w, http.StatusBadRequest, "Content or attachment is required")
		return
	}
	if attachment != "" {
		if _, err := decodeBase64Payload(attachment); err != nil {
			writeError(w, http.StatusBadRequest, "Attachment must be valid base64")
			return
		}
	}
	if len(content) > 1000 {
		writeError(w, http.StatusBadRequest, "Content exceeds maximum length")
		return
	}

	message, err := rt.db.CreateMessage(r.Context(), database.NewMessage{
		ConversationID: conversationID,
		SenderID:       ctx.AuthenticatedUser.ID,
		SenderName:     ctx.AuthenticatedUser.Name,
		Content:        content,
		Attachment:     attachment,
		ReplyTo:        replyTo,
		Timestamp:      globaltime.Now(),
	})
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusCreated, toMessageResponse(message))
}

func (rt *_router) handleForwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := strings.TrimSpace(ps.ByName("conversationId"))
	messageID := strings.TrimSpace(ps.ByName("messageId"))
	if conversationID == "" || messageID == "" {
		writeError(w, http.StatusBadRequest, "Conversation ID and message ID are required")
		return
	}
	var payload struct {
		TargetConversationID string `json:"targetConversationId"`
		TargetUserID         string `json:"targetUserId"`
		ForwarderName        string `json:"forwarderName"`
	}
	if err := decodeJSON(w, r, &payload); err != nil {
		return
	}
	payload.TargetConversationID = strings.TrimSpace(payload.TargetConversationID)
	payload.TargetUserID = strings.TrimSpace(payload.TargetUserID)
	if payload.TargetConversationID == "" && payload.TargetUserID == "" {
		writeError(w, http.StatusBadRequest, "Target conversation or user is required")
		return
	}

	targetConversationID := payload.TargetConversationID
	if targetConversationID == "" {
		convID, err := rt.db.EnsureDirectConversationID(r.Context(), ctx.AuthenticatedUser.ID, payload.TargetUserID)
		if err != nil {
			rt.respondDBError(w, ctx, err, "")
			return
		}
		targetConversationID = convID
	}

	message, err := rt.db.ForwardMessage(
		r.Context(),
		conversationID,
		messageID,
		targetConversationID,
		ctx.AuthenticatedUser.ID,
		payload.ForwarderName,
	)
	if err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	writeJSON(w, http.StatusCreated, toMessageResponse(message))
}

func (rt *_router) handleDeleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := strings.TrimSpace(ps.ByName("conversationId"))
	messageID := strings.TrimSpace(ps.ByName("messageId"))
	if conversationID == "" || messageID == "" {
		writeError(w, http.StatusBadRequest, "Conversation ID and message ID are required")
		return
	}
	if err := rt.db.DeleteMessage(r.Context(), conversationID, messageID, ctx.AuthenticatedUser.ID); err != nil {
		rt.respondDBError(w, ctx, err, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
