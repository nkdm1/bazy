package database

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/nkdm1/bazy/internal/types"
)

type MonthlyBudget struct {
	TotalAmount float64
}

// GetMonthlyPayoutBudget queries the 'payouts' table and returns
// the total amount of paid payouts within a given month.
func (db *Database) GetMonthlyPayoutBudget(year int, month time.Month) (MonthlyBudget, types.ErrorApi) {
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Second)

	row := db.queryRow(`
		SELECT COALESCE(SUM(amount), 0) AS monthly_budget
		FROM payouts
		WHERE status = 'paid'
			AND paid_at BETWEEN ? AND ?;
	`, firstDay, lastDay)

	var result MonthlyBudget
	if err := row.Scan(&result.TotalAmount); err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching monthly budget for %s %d: %v", month, year, err)
			return MonthlyBudget{}, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching monthly budget for %s %d: %v", month, year, err)
			return MonthlyBudget{}, types.ErrInternalServer
		}
	}

	return result, nil
}