package database

import (
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	sqlite3 "github.com/mattn/go-sqlite3"
)

func generateIdentifier() string {
	id, err := uuid.NewV4()
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(id.String(), "-", "")
}

func buildPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < count; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
	}
	return b.String()
}

func toInterfaceSlice(values []string) []interface{} {
	res := make([]interface{}, len(values))
	for i, v := range values {
		res[i] = v
	}
	return res
}

func sqliteIsConstraint(err error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code == sqlite3.ErrConstraint
	}
	return false
}
