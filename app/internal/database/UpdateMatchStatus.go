package database

import (
	"context"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// MarkMatchAsCompleted updates the status of a match to 'completed' by its ID.
func (db *Database) MarkMatchAsCompleted(matchID int) types.ErrorApi {
	result, err := db.exec(`
		UPDATE matches
		SET status = 'completed'
		WHERE id = ?
			AND status != 'cancelled';
	`, matchID)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while completing match %d: %v", matchID, err)
			return types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while completing match %d: %v", matchID, err)
			return types.ErrInternalServer
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[ERROR]: Could not retrieve rows affected for match %d: %v", matchID, err)
		return types.ErrInternalServer
	}
	if rowsAffected == 0 {
		return types.ErrNotFound
	}

	return nil
}