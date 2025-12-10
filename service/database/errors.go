package database

import "errors"

var (
	ErrNotFound     = errors.New("database: not found")
	ErrForbidden    = errors.New("database: forbidden")
	ErrConflict     = errors.New("database: conflict")
	ErrBadRequest   = errors.New("database: bad request")
	ErrUnauthorized = errors.New("database: unauthorized")
)
