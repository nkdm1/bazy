package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// GetPasswordHash queries the 'users' table by `email` in search of 'password_hash'
func (db *Database) GetPasswordHash(email string) (string, types.ErrorApi) {
	row := db.queryRow(`
	SELECT password_hash 
		FROM users 
		WHERE email = ? 
			AND deleted_at IS NULL;
	`, email)

	var maybeHash sql.NullString 
	if err := row.Scan(&maybeHash); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", types.ErrInvalidEmailOrPassword
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout during login: %v", err)
			return "", types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure during login: %v", err)
			return "", types.ErrInternalServer
		}
	}
	if !maybeHash.Valid {
		return "", types.ErrNullPassword
	}
	hash := maybeHash.String	
	return hash, nil
}
