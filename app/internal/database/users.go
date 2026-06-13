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

// SoftDeleteUser sets deleted_at to NOW() for the given user, effectively
// disabling their account while preserving all historical records.
func (db *Database) SoftDeleteUser(userID int) types.ErrorApi {
	result, err := db.exec(`
		UPDATE users
		SET deleted_at = NOW()
		WHERE id = ? AND deleted_at IS NULL;
	`, userID)
	if err != nil {
		log.Printf("[ERROR]: Database failure soft-deleting user %d: %v", userID, err)
		return types.ErrInternalServer
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[ERROR]: Could not retrieve rows affected for SoftDeleteUser %d: %v", userID, err)
		return types.ErrInternalServer
	}
	if rowsAffected == 0 {
		return types.ErrNotFound
	}

	return nil
}

// InvalidateAllUserSessions deletes all auth_tokens for the given user,
// immediately logging them out of every active session.
func (db *Database) InvalidateAllUserSessions(userID int) types.ErrorApi {
	_, err := db.exec(`
		DELETE FROM auth_tokens
		WHERE user_id = ?;
	`, userID)
	if err != nil {
		log.Printf("[ERROR]: Database failure invalidating sessions for user %d: %v", userID, err)
		return types.ErrInternalServer
	}
	return nil
}
