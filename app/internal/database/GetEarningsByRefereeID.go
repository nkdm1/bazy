package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

type RefereeEarnings struct {
	RefereeID    int
	TotalEarning float64
}

// GetTotalEarningsByRefereeID queries the total earnings for a referee by their ID.
// It sums amounts from 'payouts' joined through 'match_assignments',
// counting only payouts with status 'paid'.
func (db *Database) GetTotalEarningsByRefereeID(refereeID int) (RefereeEarnings, types.ErrorApi) {
	row := db.queryRow(`
		SELECT
			ma.referee_id,
			COALESCE(SUM(p.amount), 0) AS total_earnings
		FROM match_assignments ma
		JOIN payouts p
			ON p.assignment_id = ma.id
		WHERE ma.referee_id = ?
			AND p.status = 'paid'
		GROUP BY ma.referee_id;
	`, refereeID)

	var result RefereeEarnings
	if err := row.Scan(&result.RefereeID, &result.TotalEarning); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return RefereeEarnings{}, types.ErrNotFound
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching earnings for referee %d: %v", refereeID, err)
			return RefereeEarnings{}, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching earnings for referee %d: %v", refereeID, err)
			return RefereeEarnings{}, types.ErrInternalServer
		}
	}

	return result, nil
}
