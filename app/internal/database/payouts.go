package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/nkdm1/bazy/internal/types"
)

// GetMonthlyPayoutBudget queries the 'payouts' table and returns
// the total amount of paid payouts within a given month.
func (db *Database) GetMonthlyPayoutBudget(year int, month time.Month) (float64, types.ErrorApi) {
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Second)

	row, cancel := db.queryRow(`
		SELECT COALESCE(SUM(amount), 0) AS monthly_budget
		FROM payouts
		WHERE status = 'paid'
			AND paid_at BETWEEN ? AND ?;
	`, firstDay, lastDay)
	defer cancel()

	var result float64
	if err := row.Scan(&result); err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching monthly budget for %s %d: %v", month, year, err)
			return -1, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching monthly budget for %s %d: %v", month, year, err)
			return -1, types.ErrInternalServer
		}
	}

	return result, nil
}

func (db *Database) GetTotalEarningsByRefereeID(refereeID int) (float64, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT
			COALESCE(SUM(p.amount), 0) AS total_earnings
		FROM match_assignments ma
		JOIN payouts p
			ON p.assignment_id = ma.id
		WHERE ma.referee_id = ?
			AND p.status = 'paid'
	;`, refereeID)
	defer cancel()

	var result float64
	if err := row.Scan(&result); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return -1, types.ErrNotFound
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching earnings for referee %d: %v", refereeID, err)
			return -1, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching earnings for referee %d: %v", refereeID, err)
			return -1, types.ErrInternalServer
		}
	}

	return result, nil
}
