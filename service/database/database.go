package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const usersTable = "users"

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	Ping() error

	RegisterUser(ctx context.Context, name, password string) (User, string, error)
	LoginUser(ctx context.Context, name, password string) (User, string, error)
	GetUserByToken(ctx context.Context, token string) (User, error)
	RevokeToken(ctx context.Context, token string) error
	GetUserByID(ctx context.Context, id string) (User, error)
	UpdateUserPhoto(ctx context.Context, id, photo string) (User, error)
	UpdateUserName(ctx context.Context, id, name string) (User, error)
	SearchUsers(ctx context.Context, username, excludeID string, limit, offset int) ([]User, error)

	ListConversations(ctx context.Context, userID string, limit, offset int) ([]ConversationSummary, error)
	EnsureDirectConversation(ctx context.Context, requesterID, recipientID string) (ConversationDetails, error)
	EnsureDirectConversationID(ctx context.Context, requesterID, recipientID string) (string, error)
	GetConversationDetails(ctx context.Context, userID, conversationID string, messageLimit, messageOffset int) (ConversationDetails, error)

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
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			photo TEXT NOT NULL DEFAULT '',
			auth_token TEXT NOT NULL DEFAULT '',
			auth_token_expires_at TEXT NOT NULL DEFAULT '',
			password_salt TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL DEFAULT '',
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
		`CREATE TABLE IF NOT EXISTS message_status (
			message_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			delivered_at TEXT,
			read_at TEXT,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_members_user ON conversation_members(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_status_user ON message_status(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_status_message ON message_status(message_id);`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return nil, fmt.Errorf("creating database structure: %w", err)
		}
	}
	// The composite primary key already covers lookups by conversation_id.
	if _, err := db.Exec(`DROP INDEX IF EXISTS idx_members_conversation`); err != nil {
		return nil, fmt.Errorf("drop redundant members index: %w", err)
	}

	if err := ensureUsersTable(db); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_auth_token ON users(auth_token) WHERE auth_token != ''`); err != nil {
		return nil, fmt.Errorf("create auth token index: %w", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_migrations (version, name, applied_at) VALUES (1, 'baseline', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`); err != nil {
		return nil, fmt.Errorf("record baseline migration: %w", err)
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_migrations (version, name, applied_at) VALUES (2, 'secure_credentials_and_expiring_sessions', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`); err != nil {
		return nil, fmt.Errorf("record credentials migration: %w", err)
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

func ensureUsersTable(db *sql.DB) error {
	hasUsers, err := tableExists(db, usersTable)
	if err != nil {
		return err
	}
	hasUsersOld, err := tableExists(db, "users_old")
	if err != nil {
		return err
	}

	sourceTable := usersTable
	if !hasUsers && hasUsersOld {
		sourceTable = "users_old"
	}

	existing := make(map[string]struct{})
	if hasUsers || hasUsersOld {
		existing, err = tableColumns(db, sourceTable)
		if err != nil {
			return fmt.Errorf("inspect %s table: %w", sourceTable, err)
		}
	}

	required := []string{"id", "name", "photo", "auth_token", "auth_token_expires_at", "password_salt", "password_hash", "created_at"}
	needsMigration := !hasUsers
	for _, name := range required {
		if _, ok := existing[name]; !ok {
			needsMigration = true
			break
		}
	}
	if !needsMigration {
		if sourceTable != usersTable {
			needsMigration = true
		}
	}

	fkFixNeeded, err := needsUsersForeignKeyFix(db)
	if err != nil {
		return err
	}
	if !needsMigration && !fkFixNeeded {
		return nil
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = OFF;`); err != nil {
		return fmt.Errorf("disable foreign keys: %w", err)
	}
	defer func() {
		_, _ = db.Exec(`PRAGMA foreign_keys = ON;`)
	}()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin users migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`CREATE TABLE users_new (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		photo TEXT NOT NULL DEFAULT '',
		auth_token TEXT NOT NULL DEFAULT '',
		auth_token_expires_at TEXT NOT NULL DEFAULT '',
		password_salt TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);`); err != nil {
		return fmt.Errorf("create users table: %w", err)
	}

	if hasUsers || hasUsersOld {
		authSelect := "''"
		if _, ok := existing["auth_token"]; ok {
			// Legacy tokens were stored in plaintext and had no expiry. Invalidate
			// them during the security migration instead of carrying them forward.
			authSelect = "''"
		}
		authExpirySelect := "''"
		createdAtSelect := `strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`
		if _, ok := existing["created_at"]; ok {
			createdAtSelect = "created_at"
		}
		passwordSaltSelect := "''"
		if _, ok := existing["password_salt"]; ok {
			passwordSaltSelect = "password_salt"
		}
		passwordHashSelect := "''"
		if _, ok := existing["password_hash"]; ok {
			passwordHashSelect = "password_hash"
		}
		insertStmt := fmt.Sprintf(`INSERT INTO users_new (id, name, photo, auth_token, auth_token_expires_at, password_salt, password_hash, created_at)
			SELECT id, name, photo, %s, %s, %s, %s, %s FROM %s`, authSelect, authExpirySelect, passwordSaltSelect, passwordHashSelect, createdAtSelect, sourceTable)
		if _, err := tx.Exec(insertStmt); err != nil {
			return fmt.Errorf("migrate users data: %w", err)
		}
	}

	if hasUsers {
		if _, err := tx.Exec(`DROP TABLE users`); err != nil {
			return fmt.Errorf("drop old users table: %w", err)
		}
	}
	if hasUsersOld {
		if _, err := tx.Exec(`DROP TABLE users_old`); err != nil {
			return fmt.Errorf("drop legacy users table: %w", err)
		}
	}

	if _, err := tx.Exec(`ALTER TABLE users_new RENAME TO users`); err != nil {
		return fmt.Errorf("rename users table: %w", err)
	}

	if fkFixNeeded {
		if err := rebuildConversationMembers(tx); err != nil {
			return err
		}
		if err := rebuildMessages(tx); err != nil {
			return err
		}
		if err := rebuildMessageComments(tx); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit users migration: %w", err)
	}
	return nil
}

func tableExists(db *sql.DB, name string) (bool, error) {
	row := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, name)
	var table string
	err := row.Scan(&table)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("check table %s: %w", name, err)
	}
	return err == nil, nil
}

func tableColumns(db *sql.DB, table string) (map[string]struct{}, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return nil, fmt.Errorf("inspect %s table: %w", table, err)
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
			return nil, fmt.Errorf("scan %s column: %w", table, err)
		}
		_ = cid
		_ = colType
		_ = notNull
		_ = defaultV
		_ = primary
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s columns: %w", table, err)
	}
	return existing, nil
}

func needsUsersForeignKeyFix(db *sql.DB) (bool, error) {
	targets := map[string][]string{
		"conversation_members": {"user_id"},
		"messages":             {"sender_id"},
		"message_comments":     {"user_id"},
	}
	for table, userColumns := range targets {
		rows, err := db.Query(fmt.Sprintf(`PRAGMA foreign_key_list(%s)`, table))
		if err != nil {
			return false, fmt.Errorf("inspect foreign keys for %s: %w", table, err)
		}
		for rows.Next() {
			var (
				id       int
				seq      int
				refTable string
				from     string
				to       string
				onUpdate string
				onDelete string
				match    string
			)
			if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				_ = rows.Close()
				return false, fmt.Errorf("scan foreign keys for %s: %w", table, err)
			}
			_ = id
			_ = seq
			_ = to
			_ = onUpdate
			_ = onDelete
			_ = match
			for _, col := range userColumns {
				if from == col && refTable != usersTable {
					_ = rows.Close()
					return true, nil
				}
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return false, fmt.Errorf("iterate foreign keys for %s: %w", table, err)
		}
		if err := rows.Close(); err != nil {
			return false, fmt.Errorf("close foreign keys for %s: %w", table, err)
		}
	}
	return false, nil
}

func rebuildConversationMembers(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE conversation_members RENAME TO conversation_members_old`); err != nil {
		return fmt.Errorf("rename conversation_members: %w", err)
	}
	if _, err := tx.Exec(`CREATE TABLE conversation_members (
		conversation_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		added_at TEXT NOT NULL,
		PRIMARY KEY (conversation_id, user_id),
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`); err != nil {
		return fmt.Errorf("recreate conversation_members: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO conversation_members (conversation_id, user_id, added_at)
		SELECT conversation_id, user_id, added_at FROM conversation_members_old`); err != nil {
		return fmt.Errorf("migrate conversation_members: %w", err)
	}
	if _, err := tx.Exec(`DROP TABLE conversation_members_old`); err != nil {
		return fmt.Errorf("drop old conversation_members: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_members_user ON conversation_members(user_id);`); err != nil {
		return fmt.Errorf("create idx_members_user: %w", err)
	}
	return nil
}

func rebuildMessages(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE messages RENAME TO messages_old`); err != nil {
		return fmt.Errorf("rename messages: %w", err)
	}
	if _, err := tx.Exec(`CREATE TABLE messages (
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
	);`); err != nil {
		return fmt.Errorf("recreate messages: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO messages (
		id, conversation_id, sender_id, sender_name, content, attachment, created_at,
		reply_to, reply_content, reply_sender_name, reply_attachment, status
	) SELECT id, conversation_id, sender_id, sender_name, content, attachment, created_at,
		reply_to, reply_content, reply_sender_name, reply_attachment, status FROM messages_old`); err != nil {
		return fmt.Errorf("migrate messages: %w", err)
	}
	if _, err := tx.Exec(`DROP TABLE messages_old`); err != nil {
		return fmt.Errorf("drop old messages: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at DESC);`); err != nil {
		return fmt.Errorf("create idx_messages_conversation: %w", err)
	}
	return nil
}

func rebuildMessageComments(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE message_comments RENAME TO message_comments_old`); err != nil {
		return fmt.Errorf("rename message_comments: %w", err)
	}
	if _, err := tx.Exec(`CREATE TABLE message_comments (
		message_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL,
		PRIMARY KEY (message_id, user_id),
		FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`); err != nil {
		return fmt.Errorf("recreate message_comments: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO message_comments (message_id, user_id, content, created_at)
		SELECT message_id, user_id, content, created_at FROM message_comments_old`); err != nil {
		return fmt.Errorf("migrate message_comments: %w", err)
	}
	if _, err := tx.Exec(`DROP TABLE message_comments_old`); err != nil {
		return fmt.Errorf("drop old message_comments: %w", err)
	}
	return nil
}
