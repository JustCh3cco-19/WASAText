package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (db *appdbimpl) EnsureUserByName(ctx context.Context, name string) (User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return User{}, ErrBadRequest
	}

	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	user, err := scanUser(tx.QueryRowContext(ctx, `SELECT id, name, photo FROM users WHERE name = ?`, name))
	if err == nil {
		if err = tx.Commit(); err != nil {
			return User{}, fmt.Errorf("commit tx: %w", err)
		}
		return user, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, fmt.Errorf("query user: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	userID := generateIdentifier()
	if userID == "" {
		return User{}, fmt.Errorf("generate user id")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, name, photo, created_at) VALUES (?, ?, '', ?)`, userID, name, now); err != nil {
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	user = User{ID: userID, Name: name, Photo: ""}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit tx: %w", err)
	}
	return user, nil
}

func (db *appdbimpl) GetUserByID(ctx context.Context, id string) (User, error) {
	user, err := scanUser(db.c.QueryRowContext(ctx, `SELECT id, name, photo FROM users WHERE id = ?`, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("query user by id: %w", err)
	}
	return user, nil
}

func (db *appdbimpl) UpdateUserPhoto(ctx context.Context, id, photo string) (User, error) {
	res, err := db.c.ExecContext(ctx, `UPDATE users SET photo = ? WHERE id = ?`, photo, id)
	if err != nil {
		return User{}, fmt.Errorf("update user photo: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return User{}, ErrNotFound
	}
	return db.GetUserByID(ctx, id)
}

func (db *appdbimpl) UpdateUserName(ctx context.Context, id, name string) (User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return User{}, ErrBadRequest
	}

	var existing string
	err := db.c.QueryRowContext(ctx, `SELECT id FROM users WHERE name = ?`, name).Scan(&existing)
	if err == nil && existing != id {
		return User{}, ErrConflict
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, fmt.Errorf("check duplicate name: %w", err)
	}

	res, err := db.c.ExecContext(ctx, `UPDATE users SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return User{}, fmt.Errorf("update username: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return User{}, ErrNotFound
	}
	return db.GetUserByID(ctx, id)
}

func (db *appdbimpl) SearchUsers(ctx context.Context, username string) ([]User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return []User{}, nil
	}
	q := fmt.Sprintf("%%%s%%", username)

	rows, err := db.c.QueryContext(ctx,
		`SELECT id, name, photo FROM users WHERE name LIKE ? ORDER BY name LIMIT 100`,
		q)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Photo); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

func scanUser(row *sql.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Name, &u.Photo)
	return u, err
}
