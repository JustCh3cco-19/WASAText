package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestDatabase(t *testing.T) AppDatabase {
	t.Helper()
	dbConn, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	dbConn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = dbConn.Close() })
	db, err := New(dbConn)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRegisterAndLogin(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	registered, initialToken, err := db.RegisterUser(ctx, "alice", "correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	if registered.Name != "alice" || initialToken == "" {
		t.Fatalf("unexpected registration: %#v %q", registered, initialToken)
	}

	loggedIn, rotatedToken, err := db.LoginUser(ctx, "alice", "correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	if loggedIn.ID != registered.ID {
		t.Fatalf("got user %q, want %q", loggedIn.ID, registered.ID)
	}
	if rotatedToken == "" || rotatedToken == initialToken {
		t.Fatal("login did not rotate the token")
	}
	if _, err := db.GetUserByToken(ctx, initialToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old token should be invalid, got %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	if _, _, err := db.RegisterUser(ctx, "alice", "correct-horse-battery"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.LoginUser(ctx, "alice", "wrong-password"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestRegistrationRejectsDuplicateName(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	if _, _, err := db.RegisterUser(ctx, "alice", "correct-horse-battery"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.RegisterUser(ctx, "alice", "another-secure-password"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestRevokeToken(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()
	_, token, err := db.RegisterUser(ctx, "alice", "correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RevokeToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetUserByToken(ctx, token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoked token should be unauthorized, got %v", err)
	}
}
