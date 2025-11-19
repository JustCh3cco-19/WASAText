package database

import (
	"context"
	"fmt"
	"time"
)

func (db *appdbimpl) AddComment(ctx context.Context, conversationID, messageID, userID, content string) error {
	msg, err := db.getMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg.ConversationID != conversationID {
		return ErrForbidden
	}
	ok, err := db.isConversationMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}

	_, err = db.c.ExecContext(ctx, `
		INSERT INTO message_comments (message_id, user_id, content, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(message_id, user_id) DO UPDATE SET
			content=excluded.content,
			created_at=excluded.created_at`,
		messageID,
		userID,
		content,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}

func (db *appdbimpl) RemoveComment(ctx context.Context, conversationID, messageID, userID string) error {
	msg, err := db.getMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg.ConversationID != conversationID {
		return ErrForbidden
	}
	ok, err := db.isConversationMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	_, err = db.c.ExecContext(ctx, `DELETE FROM message_comments WHERE message_id = ? AND user_id = ?`, messageID, userID)
	if err != nil {
		return fmt.Errorf("remove comment: %w", err)
	}
	return nil
}
