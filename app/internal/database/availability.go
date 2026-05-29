package database

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/nkdm1/bazy/internal/types"
)

// CheckRefereeAvailability checks if a specific referee has marked themselves
// as available on a given date by querying the 'availability' table.
func (db *Database) CheckRefereeAvailability(refereeID int, date time.Time) (bool, types.ErrorApi) {
	row := db.queryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM availability
			WHERE referee_id = ?
				AND available_date = ?
		);
	`, refereeID, date.Format("2006-01-02"))

	var canBeAssigned bool
	if err := row.Scan(&canBeAssigned); err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while checking availability for referee %d: %v", refereeID, err)
			return false, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while checking availability for referee %d: %v", refereeID, err)
			return false, types.ErrInternalServer
		}
	}

	return canBeAssigned, nil
}