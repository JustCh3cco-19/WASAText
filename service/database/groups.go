package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (db *appdbimpl) CreateGroupConversation(ctx context.Context, name, photo string, memberIDs []string) (Group, error) {
	memberIDs = db.sanitizeMembers(memberIDs)
	if len(memberIDs) == 0 {
		return Group{}, ErrBadRequest
	}
	if err := db.ensureUsersExist(ctx, memberIDs); err != nil {
		return Group{}, err
	}

	groupID := generateIdentifier()
	if groupID == "" {
		return Group{}, fmt.Errorf("generate group id")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return Group{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `INSERT INTO conversations (id, name, photo, is_group, created_at) VALUES (?, ?, ?, 1, ?)`,
		groupID, name, photo, now); err != nil {
		return Group{}, fmt.Errorf("insert group conversation: %w", err)
	}

	for _, member := range memberIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id, added_at) VALUES (?, ?, ?)`, groupID, member, now); err != nil {
			return Group{}, fmt.Errorf("insert group member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Group{}, fmt.Errorf("commit tx: %w", err)
	}

	return db.buildGroup(ctx, groupID)
}

func (db *appdbimpl) ListGroups(ctx context.Context, userID string) ([]Group, error) {
	rows, err := db.c.QueryContext(ctx, `
		SELECT c.id, c.name, c.photo
		FROM conversations c
		JOIN conversation_members m ON m.conversation_id = c.id
		WHERE c.is_group = 1 AND m.user_id = ?
		ORDER BY c.name ASC, c.id ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var ids []string
	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Photo); err != nil {
			return nil, fmt.Errorf("scan group row: %w", err)
		}
		groups = append(groups, g)
		ids = append(ids, g.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups: %w", err)
	}
	if len(groups) == 0 {
		return []Group{}, nil
	}

	memberMap, err := db.loadConversationUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		users := memberMap[groups[i].ID]
		for _, u := range users {
			groups[i].Members = append(groups[i].Members, u.ID)
		}
	}
	return groups, nil
}

func (db *appdbimpl) GetGroup(ctx context.Context, userID, groupID string) (Group, error) {
	row, err := db.getGroupRow(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	isMember, err := db.isConversationMember(ctx, groupID, userID)
	if err != nil {
		return Group{}, err
	}
	if !isMember {
		return Group{}, ErrForbidden
	}
	group, err := db.buildGroup(ctx, row.ID)
	if err != nil {
		return Group{}, err
	}
	return group, nil
}

func (db *appdbimpl) LeaveGroup(ctx context.Context, userID, groupID string) error {
	if _, err := db.getGroupRow(ctx, groupID); err != nil {
		return err
	}

	res, err := db.c.ExecContext(ctx, `DELETE FROM conversation_members WHERE conversation_id = ? AND user_id = ?`, groupID, userID)
	if err != nil {
		return fmt.Errorf("leave group: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrForbidden
	}

	var membersLeft int
	if err := db.c.QueryRowContext(ctx, `SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ?`, groupID).Scan(&membersLeft); err == nil {
		if membersLeft == 0 {
			_, _ = db.c.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, groupID)
		}
	}
	return nil
}

func (db *appdbimpl) AddGroupMember(ctx context.Context, requesterID, groupID, newMemberID string) error {
	if _, err := db.getGroupRow(ctx, groupID); err != nil {
		return err
	}
	ok, err := db.isConversationMember(ctx, groupID, requesterID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	if err := db.ensureUsersExist(ctx, []string{newMemberID}); err != nil {
		return err
	}

	_, err = db.c.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id, added_at) VALUES (?, ?, ?)`, groupID, newMemberID, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		if sqliteIsConstraint(err) {
			return ErrConflict
		}
		return fmt.Errorf("add group member: %w", err)
	}
	return nil
}

func (db *appdbimpl) UpdateGroupName(ctx context.Context, requesterID, groupID, name string) (Group, error) {
	row, err := db.getGroupRow(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	ok, err := db.isConversationMember(ctx, groupID, requesterID)
	if err != nil {
		return Group{}, err
	}
	if !ok {
		return Group{}, ErrForbidden
	}

	if _, err := db.c.ExecContext(ctx, `UPDATE conversations SET name = ? WHERE id = ?`, name, row.ID); err != nil {
		return Group{}, fmt.Errorf("update group name: %w", err)
	}
	return db.buildGroup(ctx, row.ID)
}

func (db *appdbimpl) UpdateGroupPhoto(ctx context.Context, requesterID, groupID, photo string) error {
	if _, err := db.getGroupRow(ctx, groupID); err != nil {
		return err
	}
	ok, err := db.isConversationMember(ctx, groupID, requesterID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	_, err = db.c.ExecContext(ctx, `UPDATE conversations SET photo = ? WHERE id = ?`, photo, groupID)
	if err != nil {
		return fmt.Errorf("update group photo: %w", err)
	}
	return nil
}

func (db *appdbimpl) getGroupRow(ctx context.Context, groupID string) (conversationRow, error) {
	row := db.c.QueryRowContext(ctx, `SELECT id, name, photo, is_group FROM conversations WHERE id = ?`, groupID)
	var conv conversationRow
	var isGroup int
	if err := row.Scan(&conv.ID, &conv.Name, &conv.Photo, &isGroup); err != nil {
		if err == sql.ErrNoRows {
			return conversationRow{}, ErrNotFound
		}
		return conversationRow{}, fmt.Errorf("get group row: %w", err)
	}
	if isGroup != 1 {
		return conversationRow{}, ErrForbidden
	}
	return conv, nil
}

func (db *appdbimpl) buildGroup(ctx context.Context, groupID string) (Group, error) {
	row := db.c.QueryRowContext(ctx, `SELECT id, name, photo FROM conversations WHERE id = ?`, groupID)
	var g Group
	if err := row.Scan(&g.ID, &g.Name, &g.Photo); err != nil {
		return Group{}, fmt.Errorf("load group: %w", err)
	}
	memberMap, err := db.loadConversationUsers(ctx, []string{groupID})
	if err != nil {
		return Group{}, err
	}
	for _, u := range memberMap[groupID] {
		g.Members = append(g.Members, u.ID)
	}
	return g, nil
}

func (db *appdbimpl) ensureUsersExist(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return ErrBadRequest
	}
	rows, err := db.c.QueryContext(ctx, fmt.Sprintf(`SELECT id FROM users WHERE id IN (%s)`, buildPlaceholders(len(ids))), toInterfaceSlice(ids)...)
	if err != nil {
		return fmt.Errorf("check users: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	found := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan user id: %w", err)
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate user ids: %w", err)
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return ErrNotFound
		}
	}
	return nil
}
