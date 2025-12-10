package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	Ping() error

	LoginUser(ctx context.Context, name, password string) (User, string, error)
	GetUserByToken(ctx context.Context, token string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
	UpdateUserPhoto(ctx context.Context, id, photo string) (User, error)
	UpdateUserName(ctx context.Context, id, name string) (User, error)
	SearchUsers(ctx context.Context, username string) ([]User, error)

	ListConversations(ctx context.Context, userID string) ([]ConversationSummary, error)
	EnsureDirectConversation(ctx context.Context, requesterID, recipientID string) (ConversationDetails, error)
	GetConversationDetails(ctx context.Context, userID, conversationID string) (ConversationDetails, error)

	CreateGroupConversation(ctx context.Context, name, photo string, memberIDs []string) (Group, error)
	ListGroups(ctx context.Context, userID string) ([]Group, error)
	GetGroup(ctx context.Context, userID, groupID string) (Group, error)
	LeaveGroup(ctx context.Context, userID, groupID string) error
	AddGroupMember(ctx context.Context, requesterID, groupID, newMemberID string) error
	UpdateGroupName(ctx context.Context, requesterID, groupID, name string) (Group, error)
	UpdateGroupPhoto(ctx context.Context, requesterID, groupID, photo string) error

	CreateMessage(ctx context.Context, payload NewMessage) (Message, error)
	ForwardMessage(ctx context.Context, sourceConversationID, messageID, targetConversationID, requesterID, forwarderName string) (Message, error)
	DeleteMessage(ctx context.Context, conversationID, messageID, requesterID string) error

	AddComment(ctx context.Context, conversationID, messageID, userID, content string) error
	RemoveComment(ctx context.Context, conversationID, messageID, userID string) error
}

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			photo TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL DEFAULT '',
			password_salt TEXT NOT NULL DEFAULT '',
			auth_token TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			photo TEXT NOT NULL DEFAULT '',
			is_group INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS conversation_members (
			conversation_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			added_at TEXT NOT NULL,
			PRIMARY KEY (conversation_id, user_id),
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			sender_name TEXT NOT NULL,
			content TEXT NOT NULL,
			attachment TEXT NOT NULL,
			created_at TEXT NOT NULL,
			reply_to TEXT,
			reply_content TEXT,
			reply_sender_name TEXT,
			reply_attachment TEXT,
			status TEXT,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
			FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS message_comments (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_members_user ON conversation_members(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_members_conversation ON conversation_members(conversation_id);`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("creating database structure: %w", err)
		}
	}

	if err := ensureUserColumns(db); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_auth_token ON users(auth_token) WHERE auth_token != ''`); err != nil {
		return nil, fmt.Errorf("create auth token index: %w", err)
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

func (db *appdbimpl) sanitizeMembers(ids []string) []string {
	seen := make(map[string]struct{})
	res := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		res = append(res, id)
	}
	return res
}

func ensureUserColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return fmt.Errorf("inspect users table: %w", err)
	}
	defer func() { _ = rows.Close() }()

	existing := make(map[string]struct{})
	for rows.Next() {
		var (
			cid      int
			name     string
			colType  string
			notNull  int
			defaultV interface{}
			primary  int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultV, &primary); err != nil {
			return fmt.Errorf("scan users column: %w", err)
		}
		// Values are not needed beyond existence check.
		_ = cid
		_ = colType
		_ = notNull
		_ = defaultV
		_ = primary
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate users columns: %w", err)
	}

	required := map[string]string{
		"password_hash": "TEXT NOT NULL DEFAULT ''",
		"password_salt": "TEXT NOT NULL DEFAULT ''",
		"auth_token":    "TEXT NOT NULL DEFAULT ''",
	}
	for name, definition := range required {
		if _, ok := existing[name]; ok {
			continue
		}
		stmt := fmt.Sprintf(`ALTER TABLE users ADD COLUMN %s %s`, name, definition)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("add column %s: %w", name, err)
		}
	}
	return nil
}
