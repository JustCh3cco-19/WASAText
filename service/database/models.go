package database

import "time"

// User represents a WASAText user.
type User struct {
	ID    string
	Name  string
	Photo string
}

// Conversation describes a chat container.
type Conversation struct {
	ID          string
	Name        string
	Photo       string
	Members     []string
	MembersInfo []User
	IsGroup     bool
}

// ConversationSummary exposes a conversation with its last message if available.
type ConversationSummary struct {
	Conversation Conversation
	LastMessage  *Message
}

// ConversationDetails holds the full conversation plus its messages.
type ConversationDetails struct {
	Conversation Conversation
	Messages     []Message
}

// Message represents a chat message.
type Message struct {
	ID              string
	ConversationID  string
	SenderID        string
	SenderName      string
	Content         string
	Timestamp       time.Time
	Attachment      string
	ReplyTo         string
	ReplyContent    string
	ReplySenderName string
	ReplyAttachment string
	Status          string
	ReactionCount   int
	ReactingUserIDs []string
}

// NewMessage contains the information required to persist a new message.
type NewMessage struct {
	ConversationID  string
	SenderID        string
	SenderName      string
	Content         string
	Attachment      string
	ReplyTo         string
	ReplyContent    string
	ReplySenderName string
	ReplyAttachment string
	Status          string
	Timestamp       time.Time
}

// Group is a shortcut for a conversation with is_group == true.
type Group struct {
	ID          string
	Name        string
	Members     []string
	MembersInfo []User
	Photo       string
}
