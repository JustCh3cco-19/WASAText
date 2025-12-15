package api

import (
	"strings"
	"time"

	"github.com/JustCh3cco-19/WASAText/service/database"
)

type userResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Photo string `json:"photo"`
}

type messageResponse struct {
	ID              string             `json:"id"`
	SenderID        string             `json:"senderId"`
	SenderName      string             `json:"senderName"`
	Content         string             `json:"content"`
	Timestamp       string             `json:"timestamp"`
	Attachment      string             `json:"attachment"`
	ReactionCount   int                `json:"reactionCount"`
	ReactingUserIDs []string           `json:"reactingUserIds"`
	ReplyTo         string             `json:"replyTo,omitempty"`
	ReplyContent    string             `json:"replyContent,omitempty"`
	ReplySenderName string             `json:"replySenderName,omitempty"`
	ReplyAttachment string             `json:"replyAttachment,omitempty"`
	Status          string             `json:"status,omitempty"`
	Reactions       []reactionResponse `json:"reactions"`
	DeliveredTo     []string           `json:"deliveredTo,omitempty"`
	ReadBy          []string           `json:"readBy,omitempty"`
	RecipientCount  int                `json:"recipientCount,omitempty"`
	DeliveryStatus  string             `json:"deliveryStatus,omitempty"`
}

type reactionResponse struct {
	Emoji     string   `json:"emoji"`
	UserIDs   []string `json:"userIds"`
	Count     int      `json:"count"`
	UserNames []string `json:"userNames"`
}

type conversationSummaryResponse struct {
	ID                string           `json:"id"`
	Name              string           `json:"name"`
	IsGroup           bool             `json:"isGroup"`
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
	IsGroup           bool              `json:"isGroup"`
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
		Reactions:       toReactionResponses(msg.Reactions),
		DeliveredTo:     safeStringSlice(msg.DeliveredTo),
		ReadBy:          safeStringSlice(msg.ReadBy),
		RecipientCount:  msg.RecipientCount,
		DeliveryStatus:  resolveDeliveryStatus(msg),
	}
}

func toReactionResponses(rx []database.Reaction) []reactionResponse {
	if len(rx) == 0 {
		return []reactionResponse{}
	}
	grouped := make(map[string][]database.Reaction)
	for _, r := range rx {
		key := strings.TrimSpace(r.Emoji)
		if key == "" {
			key = "👍"
		}
		grouped[key] = append(grouped[key], r)
	}
	res := make([]reactionResponse, 0, len(grouped))
	for emoji, reactions := range grouped {
		var users []string
		var names []string
		for _, r := range reactions {
			users = append(users, r.UserID)
			if strings.TrimSpace(r.UserName) != "" {
				names = append(names, r.UserName)
			}
		}
		res = append(res, reactionResponse{
			Emoji:     emoji,
			UserIDs:   safeStringSlice(users),
			UserNames: safeStringSlice(names),
			Count:     len(reactions),
		})
	}
	return res
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
		IsGroup:           summary.Conversation.IsGroup,
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
		IsGroup:           details.Conversation.IsGroup,
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

func resolveDeliveryStatus(msg database.Message) string {
	if msg.RecipientCount <= 0 {
		return ""
	}
	if len(msg.ReadBy) >= msg.RecipientCount && msg.RecipientCount > 0 {
		return "read"
	}
	if len(msg.DeliveredTo) >= msg.RecipientCount {
		return "delivered"
	}
	return "sent"
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
