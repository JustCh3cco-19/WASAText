package api

import (
	"time"

	"github.com/JustCh3cco-19/WASAText/service/database"
)

type userResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Photo string `json:"photo"`
}

type messageResponse struct {
	ID              string   `json:"id"`
	SenderID        string   `json:"senderId"`
	SenderName      string   `json:"senderName"`
	Content         string   `json:"content"`
	Timestamp       string   `json:"timestamp"`
	Attachment      string   `json:"attachment"`
	ReactionCount   int      `json:"reactionCount"`
	ReactingUserIDs []string `json:"reactingUserIds"`
	ReplyTo         string   `json:"replyTo,omitempty"`
	ReplyContent    string   `json:"replyContent,omitempty"`
	ReplySenderName string   `json:"replySenderName,omitempty"`
	ReplyAttachment string   `json:"replyAttachment,omitempty"`
	Status          string   `json:"status,omitempty"`
}

type conversationSummaryResponse struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	Members           []string         `json:"members"`
	MemberDetails     []userResponse   `json:"memberDetails"`
	ConversationPhoto string           `json:"conversationPhoto"`
	LastMessage       *messageResponse `json:"lastMessage,omitempty"`
}

type conversationDetailsResponse struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Members           []string          `json:"members"`
	MemberDetails     []userResponse    `json:"memberDetails"`
	ConversationPhoto string            `json:"conversationPhoto"`
	LastMessage       *messageResponse  `json:"lastMessage,omitempty"`
	Messages          []messageResponse `json:"messages"`
}

type groupResponse struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Members       []string       `json:"members"`
	MemberDetails []userResponse `json:"memberDetails"`
	GroupPhoto    string         `json:"groupPhoto"`
}

func toUserResponse(user database.User) userResponse {
	return userResponse{
		ID:    user.ID,
		Name:  user.Name,
		Photo: user.Photo,
	}
}

func toUserResponses(users []database.User) []userResponse {
	if len(users) == 0 {
		return []userResponse{}
	}
	res := make([]userResponse, 0, len(users))
	for _, u := range users {
		res = append(res, toUserResponse(u))
	}
	return res
}

func toMessageResponse(msg database.Message) messageResponse {
	return messageResponse{
		ID:              msg.ID,
		SenderID:        msg.SenderID,
		SenderName:      msg.SenderName,
		Content:         msg.Content,
		Timestamp:       formatTimestamp(msg.Timestamp),
		Attachment:      msg.Attachment,
		ReactionCount:   msg.ReactionCount,
		ReactingUserIDs: safeStringSlice(msg.ReactingUserIDs),
		ReplyTo:         msg.ReplyTo,
		ReplyContent:    msg.ReplyContent,
		ReplySenderName: msg.ReplySenderName,
		ReplyAttachment: msg.ReplyAttachment,
		Status:          msg.Status,
	}
}

func toConversationSummaryResponse(summary database.ConversationSummary) conversationSummaryResponse {
	var last *messageResponse
	if summary.LastMessage != nil {
		msg := toMessageResponse(*summary.LastMessage)
		last = &msg
	}
	return conversationSummaryResponse{
		ID:                summary.Conversation.ID,
		Name:              summary.Conversation.Name,
		Members:           safeStringSlice(summary.Conversation.Members),
		MemberDetails:     toUserResponses(summary.Conversation.MembersInfo),
		ConversationPhoto: summary.Conversation.Photo,
		LastMessage:       last,
	}
}

func toConversationDetailsResponse(details database.ConversationDetails) conversationDetailsResponse {
	resp := conversationDetailsResponse{
		ID:                details.Conversation.ID,
		Name:              details.Conversation.Name,
		Members:           safeStringSlice(details.Conversation.Members),
		MemberDetails:     toUserResponses(details.Conversation.MembersInfo),
		ConversationPhoto: details.Conversation.Photo,
		Messages:          make([]messageResponse, 0, len(details.Messages)),
	}
	for _, msg := range details.Messages {
		resp.Messages = append(resp.Messages, toMessageResponse(msg))
	}
	if len(details.Messages) > 0 {
		last := toMessageResponse(details.Messages[len(details.Messages)-1])
		resp.LastMessage = &last
	}
	return resp
}

func toGroupResponse(gr database.Group) groupResponse {
	details := make([]userResponse, 0, len(gr.MembersInfo))
	for _, u := range gr.MembersInfo {
		details = append(details, toUserResponse(u))
	}
	return groupResponse{
		ID:            gr.ID,
		Name:          gr.Name,
		Members:       safeStringSlice(gr.Members),
		MemberDetails: details,
		GroupPhoto:    gr.Photo,
	}
}

func safeStringSlice(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
