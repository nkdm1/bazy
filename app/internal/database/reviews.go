package database

import (
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// RateRefereePerformance adds a review for a referee's performance in a specific match.
// It verifies that the match status is 'completed' before inserting the review.
func (db *Database) RateRefereePerformance(refereeID, matchID, rating, createdBy int) types.ErrorApi {
	row, cancel := db.queryRow(`
		SELECT status
		FROM matches
		WHERE id = ?;
	`, matchID)
	defer cancel()

	var status string
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		log.Printf("[ERROR]: Database failure while fetching match %d: %v", matchID, err)
		return types.ErrInternalServer
	}

	if status != "completed" {
		return types.ErrInvalidPayload // Or a more specific error like match not finished
	}

	_, err := db.exec(`
		INSERT INTO reviews (referee_id, match_id, rating, created_by)
		VALUES (?, ?, ?, ?);
	`, refereeID, matchID, rating, createdBy)

	if err != nil {
		log.Printf("[ERROR]: Database failure while inserting review: %v", err)
		return types.ErrInternalServer
	}

	return nil
}
