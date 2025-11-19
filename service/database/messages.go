package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (db *appdbimpl) CreateMessage(ctx context.Context, payload NewMessage) (Message, error) {
	ok, err := db.isConversationMember(ctx, payload.ConversationID, payload.SenderID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrForbidden
	}

	messageID := generateIdentifier()
	if messageID == "" {
		return Message{}, fmt.Errorf("generate message id")
	}
	createdAt := payload.Timestamp.UTC().Format(time.RFC3339Nano)

	_, err = db.c.ExecContext(ctx, `
		INSERT INTO messages (
			id, conversation_id, sender_id, sender_name, content, attachment, created_at,
			reply_to, reply_content, reply_sender_name, reply_attachment, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		messageID,
		payload.ConversationID,
		payload.SenderID,
		payload.SenderName,
		payload.Content,
		payload.Attachment,
		createdAt,
		nullIfEmpty(payload.ReplyTo),
		nullIfEmpty(payload.ReplyContent),
		nullIfEmpty(payload.ReplySenderName),
		nullIfEmpty(payload.ReplyAttachment),
		nullIfEmpty(payload.Status),
	)
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}

	return db.getMessageByID(ctx, messageID)
}

func (db *appdbimpl) ForwardMessage(ctx context.Context, sourceConversationID, messageID, targetConversationID, requesterID, forwarderName string) (Message, error) {
	ok, err := db.isConversationMember(ctx, sourceConversationID, requesterID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrForbidden
	}
	ok, err = db.isConversationMember(ctx, targetConversationID, requesterID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{}, ErrForbidden
	}

	original, err := db.getMessageByID(ctx, messageID)
	if err != nil {
		return Message{}, err
	}
	if original.ConversationID != sourceConversationID {
		return Message{}, ErrForbidden
	}

	sender, err := db.GetUserByID(ctx, requesterID)
	if err != nil {
		return Message{}, err
	}

	name := sender.Name
	if trimmed := strings.TrimSpace(forwarderName); trimmed != "" {
		name = trimmed
	}

	return db.CreateMessage(ctx, NewMessage{
		ConversationID:  targetConversationID,
		SenderID:        requesterID,
		SenderName:      name,
		Content:         original.Content,
		Attachment:      original.Attachment,
		ReplyTo:         original.ReplyTo,
		ReplyContent:    original.ReplyContent,
		ReplySenderName: original.ReplySenderName,
		ReplyAttachment: original.ReplyAttachment,
		Status:          original.Status,
		Timestamp:       time.Now().UTC(),
	})
}

func (db *appdbimpl) DeleteMessage(ctx context.Context, conversationID, messageID, requesterID string) error {
	msg, err := db.getMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg.ConversationID != conversationID {
		return ErrForbidden
	}
	if msg.SenderID != requesterID {
		return ErrForbidden
	}
	res, err := db.c.ExecContext(ctx, `DELETE FROM messages WHERE id = ?`, messageID)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *appdbimpl) getMessageByID(ctx context.Context, messageID string) (Message, error) {
	row := db.c.QueryRowContext(ctx, `
		SELECT id, conversation_id, sender_id, sender_name, content, attachment, created_at,
		       reply_to, reply_content, reply_sender_name, reply_attachment, status
		FROM messages WHERE id = ?`, messageID)
	var msg Message
	var createdAt string
	if err := row.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.SenderName, &msg.Content, &msg.Attachment,
		&createdAt, &msg.ReplyTo, &msg.ReplyContent, &msg.ReplySenderName, &msg.ReplyAttachment, &msg.Status); err != nil {
		if err == sql.ErrNoRows {
			return Message{}, ErrNotFound
		}
		return Message{}, fmt.Errorf("get message: %w", err)
	}
	if createdAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
			msg.Timestamp = parsed
		}
	}
	reactions, err := db.loadReactions(ctx, []string{msg.ID})
	if err == nil {
		msg.ReactingUserIDs = reactions[msg.ID]
		msg.ReactionCount = len(msg.ReactingUserIDs)
	}
	return msg, nil
}

func nullIfEmpty(v string) interface{} {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
