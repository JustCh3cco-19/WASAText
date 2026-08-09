// Package application is the boundary for WASAText use cases. HTTP handlers
// depend on this package rather than directly on a concrete storage engine.
package application

import (
	"errors"

	"github.com/JustCh3cco-19/WASAText/service/database"
)

// Service exposes the application use cases. Embedding the storage contract
// keeps the first migration small while allowing business rules to move here
// incrementally without coupling HTTP handlers to SQLite.
type Service interface {
	database.AppDatabase
}

type service struct {
	database.AppDatabase
}

func New(db database.AppDatabase) (Service, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	return &service{AppDatabase: db}, nil
}
