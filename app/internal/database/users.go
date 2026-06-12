package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// GetUserRole fetches the role of a user from the users table.
func (db *Database) GetUserRole(userID int) (string, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT role
		FROM users
		WHERE id = ?;
	`, userID)
	defer cancel()

	var role string
	if err := row.Scan(&role); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", types.ErrNotFound
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching user role %d: %v", userID, err)
			return "", types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching user role %d: %v", userID, err)
			return "", types.ErrInternalServer
		}
	}
	return role, nil
}
