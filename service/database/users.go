package database

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	passwordSaltBytes  = 16
	passwordIterations = 210000
	passwordKeyBytes   = 32
	sessionDuration    = 24 * time.Hour
)

func (db *appdbimpl) RegisterUser(ctx context.Context, name, password string) (User, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) < 3 || len(name) > 16 || !validPassword(password) {
		return User{}, "", ErrBadRequest
	}
	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return User{}, "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	userID, err := generateNonEmptyIdentifier()
	if err != nil {
		return User{}, "", err
	}
	token, err := generateNonEmptyIdentifier()
	if err != nil {
		return User{}, "", err
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return User{}, "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := pbkdf2SHA256([]byte(password), salt, passwordIterations, passwordKeyBytes)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	expiresAt := time.Now().UTC().Add(sessionDuration).Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `INSERT INTO users (id, name, photo, auth_token, auth_token_expires_at, password_salt, password_hash, created_at) VALUES (?, ?, '', ?, ?, ?, ?, ?)`,
		userID, name, hashToken(token), expiresAt, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash), now)
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
}

func (db *appdbimpl) LoginUser(ctx context.Context, name, password string) (User, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || password == "" {
		return User{}, "", ErrUnauthorized
	}
	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return User{}, "", fmt.Errorf("begin login: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var user User
	var saltEncoded, hashEncoded string
	err = tx.QueryRowContext(ctx, `SELECT id, name, photo, password_salt, password_hash FROM users WHERE name = ?`, name).
		Scan(&user.ID, &user.Name, &user.Photo, &saltEncoded, &hashEncoded)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, "", ErrUnauthorized
	}
	if err != nil {
		return User{}, "", fmt.Errorf("query user: %w", err)
	}
	salt, saltErr := base64.RawStdEncoding.DecodeString(saltEncoded)
	want, hashErr := base64.RawStdEncoding.DecodeString(hashEncoded)
	if saltErr != nil || hashErr != nil || len(salt) == 0 || len(want) != passwordKeyBytes ||
		!hmac.Equal(pbkdf2SHA256([]byte(password), salt, passwordIterations, passwordKeyBytes), want) {
		return User{}, "", ErrUnauthorized
	}
	token, err := db.rotateToken(ctx, tx, user.ID)
	if err != nil {
		return User{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return User{}, "", fmt.Errorf("commit login: %w", err)
	}
	return user, token, nil
}

func validPassword(password string) bool { return len(password) >= 10 && len(password) <= 128 }

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	result := make([]byte, 0, keyLen)
	for block := uint32(1); len(result) < keyLen; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
	}
	return result[:keyLen]
}

func (db *appdbimpl) GetUserByToken(ctx context.Context, token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, ErrUnauthorized
	}
	user, err := scanUser(db.c.QueryRowContext(ctx, `SELECT id, name, photo FROM users WHERE auth_token = ? AND auth_token_expires_at > ?`, hashToken(token), time.Now().UTC().Format(time.RFC3339Nano)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUnauthorized
		}
		return User{}, fmt.Errorf("query user by token: %w", err)
	}
	return user, nil
}

func (db *appdbimpl) RevokeToken(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrUnauthorized
	}
	res, err := db.c.ExecContext(ctx, `UPDATE users SET auth_token = '', auth_token_expires_at = '' WHERE auth_token = ?`, hashToken(token))
	if err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke token rows: %w", err)
	}
	if affected == 0 {
		return ErrUnauthorized
	}
	return nil
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

	tx, err := db.c.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin update username: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existing string
	err = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE name = ?`, name).Scan(&existing)
	if err == nil && existing != id {
		return User{}, ErrConflict
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, fmt.Errorf("check duplicate name: %w", err)
	}

	res, err := tx.ExecContext(ctx, `UPDATE users SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		if sqliteIsConstraint(err) {
			return User{}, ErrConflict
		}
		return User{}, fmt.Errorf("update username: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return User{}, ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `UPDATE messages SET sender_name = ? WHERE sender_id = ?`, name, id); err != nil {
		return User{}, fmt.Errorf("update message sender names: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE messages SET reply_sender_name = ? WHERE reply_to IN (SELECT id FROM messages WHERE sender_id = ?)`, name, id); err != nil {
		return User{}, fmt.Errorf("update reply sender names: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit update username: %w", err)
	}
	return db.GetUserByID(ctx, id)
}

func (db *appdbimpl) SearchUsers(ctx context.Context, username, excludeID string, limit, offset int) ([]User, error) {
	username = strings.TrimSpace(username)
	excludeID = strings.TrimSpace(excludeID)
	if limit < 1 || limit > 200 || offset < 0 {
		return nil, ErrBadRequest
	}
	if username == "" {
		args := []interface{}{}
		query := `SELECT id, name, photo FROM users`
		if excludeID != "" {
			query += ` WHERE id != ?`
			args = append(args, excludeID)
		}
		query += ` ORDER BY name LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
		rows, err := db.c.QueryContext(ctx,
			query, args...)
		if err != nil {
			return nil, fmt.Errorf("search users: %w", err)
		}
		defer func() { _ = rows.Close() }()

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
	q := fmt.Sprintf("%%%s%%", username)
	args := []interface{}{q}
	query := `SELECT id, name, photo FROM users WHERE name LIKE ?`
	if excludeID != "" {
		query += ` AND id != ?`
		args = append(args, excludeID)
	}
	query += ` ORDER BY name LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.c.QueryContext(ctx, query, args...)
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
	expiresAt := time.Now().UTC().Add(sessionDuration).Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE users SET auth_token = ?, auth_token_expires_at = ? WHERE id = ?`, hashToken(token), expiresAt, userID); err != nil {
		return "", fmt.Errorf("rotate token: %w", err)
	}
	return token, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func generateNonEmptyIdentifier() (string, error) {
	id := generateIdentifier()
	if id == "" {
		return "", fmt.Errorf("generate identifier")
	}
	return id, nil
}
