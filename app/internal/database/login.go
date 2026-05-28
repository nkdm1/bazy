package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

func (db *Database) GetPasswordHash(email string) (string, types.ErrorApi) {
	row := db.queryRow(`
	SELECT id, password_hash 
		FROM users 
		WHERE email = ? 
			AND deleted_at IS NULL;
	`, email)

	var hash string
	if err := row.Scan(&hash); err != nil {
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
	return hash, nil
}
