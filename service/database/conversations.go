package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"
)

type conversationRow struct {
	ID      string
	Name    string
	Photo   string
	IsGroup bool
}

func (db *appdbimpl) ListConversations(ctx context.Context, userID string) ([]ConversationSummary, error) {
	if err := db.markMessagesDeliveredForUser(ctx, userID); err != nil {
		return nil, err
	}

	rows, err := db.c.QueryContext(ctx, `
		SELECT c.id, c.name, c.photo, c.is_group
		FROM conversations c
		JOIN conversation_members m ON m.conversation_id = c.id
		WHERE m.user_id = ?
		ORDER BY (
			SELECT COALESCE(MAX(created_at), c.created_at) FROM messages WHERE conversation_id = c.id
		) DESC, c.created_at DESC, c.id ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var convRows []conversationRow
	for rows.Next() {
		var r conversationRow
		var isGroup int
		if err := rows.Scan(&r.ID, &r.Name, &r.Photo, &isGroup); err != nil {
			return nil, fmt.Errorf("scan conversation row: %w", err)
		}
		r.IsGroup = isGroup == 1
		convRows = append(convRows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}
	if len(convRows) == 0 {
		return []ConversationSummary{}, nil
	}

	for _, c := range convRows {
		if err := db.ensureConversationStatuses(ctx, c.ID); err != nil {
			return nil, err
		}
	}

	ids := make([]string, len(convRows))
	for i, c := range convRows {
		ids[i] = c.ID
	}

	memberMap, err := db.loadConversationUsers(ctx, ids)
	if err != nil {
		return nil, err
	}

	lastMessages, err := db.loadLastMessages(ctx, ids)
	if err != nil {
		return nil, err
	}

	summaries := make([]ConversationSummary, 0, len(convRows))
	for _, row := range convRows {
		members := memberMap[row.ID]
		conv := db.buildConversationForViewer(row, members, userID)
		summaries = append(summaries, ConversationSummary{
			Conversation: conv,
			LastMessage:  lastMessages[row.ID],
		})
	}

	return summaries, nil
}

func (db *appdbimpl) ensureDirectConversationID(ctx context.Context, requesterID, recipientID string) (string, error) {
	if requesterID == recipientID {
		return "", ErrBadRequest
	}

	if _, err := db.GetUserByID(ctx, recipientID); err != nil {
		return "", err
	}

	var conversationID string
	err := db.c.QueryRowContext(ctx, `
		SELECT c.id
		FROM conversations c
		WHERE c.is_group = 0
		  AND EXISTS (SELECT 1 FROM conversation_members m1 WHERE m1.conversation_id = c.id AND m1.user_id = ?)
		  AND EXISTS (SELECT 1 FROM conversation_members m2 WHERE m2.conversation_id = c.id AND m2.user_id = ?)
		  AND (SELECT COUNT(*) FROM conversation_members m WHERE m.conversation_id = c.id) = 2
		LIMIT 1`, requesterID, recipientID).Scan(&conversationID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("query direct conversation: %w", err)
	}
	if err == nil {
		return conversationID, nil
	}

	conversationID = generateIdentifier()
	if conversationID == "" {
		return "", fmt.Errorf("generate conversation id")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `INSERT INTO conversations (id, name, photo, is_group, created_at) VALUES (?, '', '', 0, ?)`, conversationID, now); err != nil {
		return "", fmt.Errorf("insert conversation: %w", err)
	}

	members := []string{requesterID, recipientID}
	for _, member := range members {
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id, added_at) VALUES (?, ?, ?)`, conversationID, member, now); err != nil {
			return "", fmt.Errorf("insert conversation member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}

	return conversationID, nil
}

func (db *appdbimpl) EnsureDirectConversationID(ctx context.Context, requesterID, recipientID string) (string, error) {
	return db.ensureDirectConversationID(ctx, requesterID, recipientID)
}

func (db *appdbimpl) EnsureDirectConversation(ctx context.Context, requesterID, recipientID string) (ConversationDetails, error) {
	conversationID, err := db.ensureDirectConversationID(ctx, requesterID, recipientID)
	if err != nil {
		return ConversationDetails{}, err
	}
	return db.GetConversationDetails(ctx, requesterID, conversationID)
}

func (db *appdbimpl) GetConversationDetails(ctx context.Context, userID, conversationID string) (ConversationDetails, error) {
	row := db.c.QueryRowContext(ctx, `SELECT id, name, photo, is_group FROM conversations WHERE id = ?`, conversationID)
	var convRow conversationRow
	var isGroup int
	if err := row.Scan(&convRow.ID, &convRow.Name, &convRow.Photo, &isGroup); err != nil {
		if err == sql.ErrNoRows {
			return ConversationDetails{}, ErrNotFound
		}
		return ConversationDetails{}, fmt.Errorf("load conversation: %w", err)
	}
	convRow.IsGroup = isGroup == 1

	isMember, err := db.isConversationMember(ctx, conversationID, userID)
	if err != nil {
		return ConversationDetails{}, err
	}
	if !isMember {
		return ConversationDetails{}, ErrForbidden
	}

	if err := db.ensureConversationStatuses(ctx, conversationID); err != nil {
		return ConversationDetails{}, err
	}
	if err := db.markMessagesDeliveredForConversation(ctx, conversationID, userID); err != nil {
		return ConversationDetails{}, err
	}
	if err := db.markMessagesReadForConversation(ctx, conversationID, userID); err != nil {
		return ConversationDetails{}, err
	}

	memberMap, err := db.loadConversationUsers(ctx, []string{conversationID})
	if err != nil {
		return ConversationDetails{}, err
	}

	messages, err := db.loadMessagesForConversation(ctx, conversationID)
	if err != nil {
		return ConversationDetails{}, err
	}

	conv := db.buildConversationForViewer(convRow, memberMap[conversationID], userID)
	return ConversationDetails{
		Conversation: conv,
		Messages:     messages,
	}, nil
}

func (db *appdbimpl) loadConversationUsers(ctx context.Context, conversationIDs []string) (map[string][]User, error) {
	result := make(map[string][]User, len(conversationIDs))
	if len(conversationIDs) == 0 {
		return result, nil
	}

	query := fmt.Sprintf(
		`SELECT cm.conversation_id, u.id, u.name, u.photo
		 FROM conversation_members cm
		 JOIN users u ON u.id = cm.user_id
		 WHERE cm.conversation_id IN (%s)
		 ORDER BY cm.conversation_id, u.id`,
		buildPlaceholders(len(conversationIDs)),
	)
	rows, err := db.c.QueryContext(ctx, query, toInterfaceSlice(conversationIDs)...)
	if err != nil {
		return nil, fmt.Errorf("load conversation members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var conversationID string
		var u User
		if err := rows.Scan(&conversationID, &u.ID, &u.Name, &u.Photo); err != nil {
			return nil, fmt.Errorf("scan conversation user: %w", err)
		}
		result[conversationID] = append(result[conversationID], u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation users: %w", err)
	}

	for id, users := range result {
		sort.Slice(users, func(i, j int) bool {
			return users[i].ID < users[j].ID
		})
		result[id] = users
	}

	return result, nil
}

func (db *appdbimpl) buildConversationForViewer(row conversationRow, users []User, viewerID string) Conversation {
	conv := Conversation{
		ID:          row.ID,
		Name:        row.Name,
		Photo:       row.Photo,
		IsGroup:     row.IsGroup,
		Members:     nil,
		MembersInfo: nil,
	}
	for _, u := range users {
		conv.Members = append(conv.Members, u.ID)
		conv.MembersInfo = append(conv.MembersInfo, u)
	}

	if row.IsGroup {
		return conv
	}

	for _, u := range users {
		if u.ID != viewerID {
			conv.Name = u.Name
			conv.Photo = u.Photo
			break
		}
	}
	if conv.Name == "" && len(users) > 0 {
		conv.Name = users[0].Name
		conv.Photo = users[0].Photo
	}
	return conv
}

func (db *appdbimpl) loadLastMessages(ctx context.Context, conversationIDs []string) (map[string]*Message, error) {
	result := make(map[string]*Message, len(conversationIDs))
	if len(conversationIDs) == 0 {
		return result, nil
	}

	query := fmt.Sprintf(`
		SELECT m1.id, m1.conversation_id, m1.sender_id, m1.sender_name, m1.content, m1.attachment,
		       m1.created_at, m1.reply_to, m1.reply_content, m1.reply_sender_name, m1.reply_attachment, m1.status
		FROM messages m1
		JOIN (
			SELECT conversation_id, MAX(created_at) AS max_created
			FROM messages
			WHERE conversation_id IN (%s)
			GROUP BY conversation_id
		) latest ON latest.conversation_id = m1.conversation_id AND latest.max_created = m1.created_at`,
		buildPlaceholders(len(conversationIDs)),
	)
	rows, err := db.c.QueryContext(ctx, query, toInterfaceSlice(conversationIDs)...)
	if err != nil {
		return nil, fmt.Errorf("load last messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messageIDs []string
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan last message: %w", err)
		}
		result[msg.ConversationID] = &msg
		messageIDs = append(messageIDs, msg.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate last messages: %w", err)
	}

	if len(messageIDs) > 0 {
		receipts, err := db.loadMessageReceipts(ctx, messageIDs)
		if err != nil {
			return nil, err
		}
		reactions, err := db.loadReactions(ctx, messageIDs)
		if err != nil {
			return nil, err
		}
		for _, msg := range result {
			if rx, ok := reactions[msg.ID]; ok {
				msg.Reactions = rx
				msg.ReactingUserIDs = uniqueReactionUsers(rx)
				msg.ReactionCount = len(msg.ReactingUserIDs)
			}
			if rcpts, ok := receipts[msg.ID]; ok {
				for _, r := range rcpts {
					if r.Delivered {
						msg.DeliveredTo = append(msg.DeliveredTo, r.UserID)
					}
					if r.Read {
						msg.ReadBy = append(msg.ReadBy, r.UserID)
					}
				}
				msg.RecipientCount = len(rcpts)
			}
		}
	}

	return result, nil
}

func (db *appdbimpl) isConversationMember(ctx context.Context, conversationID, userID string) (bool, error) {
	var exists int
	err := db.c.QueryRowContext(ctx, `SELECT 1 FROM conversation_members WHERE conversation_id = ? AND user_id = ?`, conversationID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check conversation membership: %w", err)
	}
	return true, nil
}

func (db *appdbimpl) loadMessagesForConversation(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := db.c.QueryContext(ctx, `
		SELECT id, conversation_id, sender_id, sender_name, content, attachment, created_at,
		       reply_to, reply_content, reply_sender_name, reply_attachment, status
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var messages []Message
	var messageIDs []string
	for rows.Next() {
		msg, err := scanMessageRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}
		messages = append(messages, msg)
		messageIDs = append(messageIDs, msg.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	if len(messageIDs) == 0 {
		return []Message{}, nil
	}

	reactions, err := db.loadReactions(ctx, messageIDs)
	if err != nil {
		return nil, err
	}
	receipts, err := db.loadMessageReceipts(ctx, messageIDs)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		if rx, ok := reactions[messages[i].ID]; ok {
			messages[i].Reactions = rx
			messages[i].ReactingUserIDs = uniqueReactionUsers(rx)
			messages[i].ReactionCount = len(messages[i].ReactingUserIDs)
		}
		if rcpts, ok := receipts[messages[i].ID]; ok {
			for _, r := range rcpts {
				if r.Delivered {
					messages[i].DeliveredTo = append(messages[i].DeliveredTo, r.UserID)
				}
				if r.Read {
					messages[i].ReadBy = append(messages[i].ReadBy, r.UserID)
				}
			}
			messages[i].RecipientCount = len(rcpts)
		}
	}

	return messages, nil
}

func scanMessageRow(rows *sql.Rows) (Message, error) {
	var (
		timestamp       string
		replyTo         sql.NullString
		replyContent    sql.NullString
		replySenderName sql.NullString
		replyAttachment sql.NullString
		status          sql.NullString
		msg             Message
	)
	err := rows.Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderID,
		&msg.SenderName,
		&msg.Content,
		&msg.Attachment,
		&timestamp,
		&replyTo,
		&replyContent,
		&replySenderName,
		&replyAttachment,
		&status,
	)
	if err != nil {
		return Message{}, err
	}
	if timestamp != "" {
		if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
			msg.Timestamp = t
		}
	}
	if replyTo.Valid {
		msg.ReplyTo = replyTo.String
	}
	if replyContent.Valid {
		msg.ReplyContent = replyContent.String
	}
	if replySenderName.Valid {
		msg.ReplySenderName = replySenderName.String
	}
	if replyAttachment.Valid {
		msg.ReplyAttachment = replyAttachment.String
	}
	if status.Valid {
		msg.Status = status.String
	}
	return msg, nil
}

func (db *appdbimpl) loadReactions(ctx context.Context, messageIDs []string) (map[string][]Reaction, error) {
	result := make(map[string][]Reaction, len(messageIDs))
	if len(messageIDs) == 0 {
		return result, nil
	}
	query := fmt.Sprintf(`SELECT mc.message_id, mc.user_id, u.name, mc.content
		FROM message_comments mc
		JOIN users u ON u.id = mc.user_id
		WHERE mc.message_id IN (%s)
		ORDER BY mc.created_at ASC`, buildPlaceholders(len(messageIDs)))
	rows, err := db.c.QueryContext(ctx, query, toInterfaceSlice(messageIDs)...)
	if err != nil {
		return nil, fmt.Errorf("load reactions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var messageID, userID, userName, emoji string
		if err := rows.Scan(&messageID, &userID, &userName, &emoji); err != nil {
			return nil, fmt.Errorf("scan reaction: %w", err)
		}
		result[messageID] = append(result[messageID], Reaction{
			Emoji:    emoji,
			UserID:   userID,
			UserName: userName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reactions: %w", err)
	}
	return result, nil
}

func uniqueReactionUsers(rx []Reaction) []string {
	seen := make(map[string]struct{})
	users := make([]string, 0, len(rx))
	for _, r := range rx {
		if _, ok := seen[r.UserID]; ok {
			continue
		}
		seen[r.UserID] = struct{}{}
		users = append(users, r.UserID)
	}
	return users
}
