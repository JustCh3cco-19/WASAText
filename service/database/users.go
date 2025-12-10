package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (db *appdbimpl) LoginUser(ctx context.Context, name string) (User, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) < 3 || len(name) > 16 {
		return User{}, "", ErrBadRequest
	}

	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return User{}, "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var user User
	err = tx.QueryRowContext(ctx, `SELECT id, name, photo FROM users WHERE name = ?`, name).
		Scan(&user.ID, &user.Name, &user.Photo)
	switch {
	case err == nil:
		token, err := db.rotateToken(ctx, tx, user.ID)
		if err != nil {
			return User{}, "", err
		}
		if err := tx.Commit(); err != nil {
			return User{}, "", fmt.Errorf("commit tx: %w", err)
		}
		return user, token, nil
	case errors.Is(err, sql.ErrNoRows):
		userID, err := generateNonEmptyIdentifier()
		if err != nil {
			return User{}, "", err
		}
		token, err := generateNonEmptyIdentifier()
		if err != nil {
			return User{}, "", err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_, err = tx.ExecContext(ctx, `INSERT INTO users (id, name, photo, auth_token, created_at) VALUES (?, ?, '', ?, ?)`,
			userID, name, token, now)
		if err != nil {
			if sqliteIsConstraint(err) {
				return User{}, "", ErrConflict
			}
			return User{}, "", fmt.Errorf("insert user: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return User{}, "", fmt.Errorf("commit tx: %w", err)
		}
		return User{ID: userID, Name: name, Photo: ""}, token, nil
	default:
		return User{}, "", fmt.Errorf("query user: %w", err)
	}
}

func (db *appdbimpl) GetUserByToken(ctx context.Context, token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, ErrUnauthorized
	}
	user, err := scanUser(db.c.QueryRowContext(ctx, `SELECT id, name, photo FROM users WHERE auth_token = ?`, token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUnauthorized
		}
		return User{}, fmt.Errorf("query user by token: %w", err)
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

func (db *appdbimpl) rotateToken(ctx context.Context, tx *sql.Tx, userID string) (string, error) {
	token, err := generateNonEmptyIdentifier()
	if err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET auth_token = ? WHERE id = ?`, token, userID); err != nil {
		return "", fmt.Errorf("rotate token: %w", err)
	}
	return token, nil
}

func generateNonEmptyIdentifier() (string, error) {
	id := generateIdentifier()
	if id == "" {
		return "", fmt.Errorf("generate identifier")
	}
	return id, nil
}
