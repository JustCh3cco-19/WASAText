package database

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCriticalQueriesUseIndexes(t *testing.T) {
	conn, err := sql.Open("sqlite3", "file:query-plan?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := New(conn); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, query, index string
		args               []interface{}
	}{
		{"conversation messages", `SELECT id FROM messages WHERE conversation_id = ? ORDER BY created_at DESC`, "idx_messages_conversation", []interface{}{"conversation"}},
		{"user receipts", `SELECT message_id FROM message_status WHERE user_id = ?`, "idx_status_user", []interface{}{"user"}},
		{"conversation members", `SELECT user_id FROM conversation_members WHERE conversation_id = ?`, "sqlite_autoindex_conversation_members", []interface{}{"conversation"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rows, err := conn.Query("EXPLAIN QUERY PLAN "+test.query, test.args...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rows.Close() }()
			var plan strings.Builder
			for rows.Next() {
				var id, parent, unused int
				var detail string
				if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
					t.Fatal(err)
				}
				plan.WriteString(detail)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(plan.String(), test.index) {
				t.Fatalf("query plan %q does not use %s", plan.String(), test.index)
			}
		})
	}
}
