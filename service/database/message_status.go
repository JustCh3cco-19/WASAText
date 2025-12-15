package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type messageReceipt struct {
	MessageID string
	UserID    string
	Delivered bool
	Read      bool
}

func (db *appdbimpl) markMessagesDeliveredForUser(ctx context.Context, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.c.ExecContext(ctx, `UPDATE message_status SET delivered_at = ? WHERE user_id = ? AND delivered_at IS NULL`, now, userID)
	if err != nil {
		return fmt.Errorf("mark delivered for user: %w", err)
	}
	return nil
}

func (db *appdbimpl) markMessagesDeliveredForConversation(ctx context.Context, conversationID, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.c.ExecContext(ctx, `
		UPDATE message_status
		SET delivered_at = ?
		WHERE user_id = ?
		  AND delivered_at IS NULL
		  AND message_id IN (SELECT id FROM messages WHERE conversation_id = ?)`,
		now, userID, conversationID)
	if err != nil {
		return fmt.Errorf("mark delivered for conversation: %w", err)
	}
	return nil
}

func (db *appdbimpl) markMessagesReadForConversation(ctx context.Context, conversationID, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := db.c.ExecContext(ctx, `
		UPDATE message_status
		SET read_at = ?, delivered_at = COALESCE(delivered_at, ?)
		WHERE user_id = ?
		  AND read_at IS NULL
		  AND message_id IN (SELECT id FROM messages WHERE conversation_id = ?)`,
		now, now, userID, conversationID)
	if err != nil {
		return fmt.Errorf("mark read for conversation: %w", err)
	}
	return nil
}

func (db *appdbimpl) loadMessageReceipts(ctx context.Context, messageIDs []string) (map[string][]messageReceipt, error) {
	result := make(map[string][]messageReceipt, len(messageIDs))
	if len(messageIDs) == 0 {
		return result, nil
	}
	query := fmt.Sprintf(
		`SELECT message_id, user_id, delivered_at, read_at
		 FROM message_status
		 WHERE message_id IN (%s)
		 ORDER BY message_id, user_id`,
		buildPlaceholders(len(messageIDs)),
	)
	rows, err := db.c.QueryContext(ctx, query, toInterfaceSlice(messageIDs)...)
	if err != nil {
		return nil, fmt.Errorf("load message receipts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			msgID     string
			userID    string
			delivered sql.NullString
			read      sql.NullString
			receipt   messageReceipt
		)
		if err := rows.Scan(&msgID, &userID, &delivered, &read); err != nil {
			return nil, fmt.Errorf("scan message receipt: %w", err)
		}
		receipt = messageReceipt{
			MessageID: msgID,
			UserID:    userID,
			Delivered: delivered.Valid && delivered.String != "",
			Read:      read.Valid && read.String != "",
		}
		result[msgID] = append(result[msgID], receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message receipts: %w", err)
	}
	return result, nil
}

func (db *appdbimpl) getConversationMemberIDs(ctx context.Context, conversationID string) ([]string, error) {
	rows, err := db.c.QueryContext(ctx, `SELECT user_id FROM conversation_members WHERE conversation_id = ?`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var members []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan conversation member: %w", err)
		}
		members = append(members, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation members: %w", err)
	}
	return members, nil
}

func (db *appdbimpl) ensureConversationStatuses(ctx context.Context, conversationID string) error {
	_, err := db.c.ExecContext(ctx, `
		INSERT OR IGNORE INTO message_status (message_id, user_id, delivered_at, read_at)
		SELECT m.id, cm.user_id, NULL, NULL
		FROM messages m
		JOIN conversation_members cm ON cm.conversation_id = m.conversation_id
		WHERE m.conversation_id = ?
		  AND m.sender_id != cm.user_id`,
		conversationID)
	if err != nil {
		return fmt.Errorf("ensure message statuses: %w", err)
	}
	return nil
}
