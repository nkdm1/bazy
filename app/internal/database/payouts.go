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

// CreatePendingPayout resolves applicable wage and inserts 'pending' payout
func (db *Database) CreatePendingPayout(assignmentID int) types.ErrorApi {
	row, cancel := db.queryRow(`
		SELECT m.level_of_match, ma.role, m.match_start
		FROM match_assignments ma
		JOIN matches m ON ma.match_id = m.id
		WHERE ma.id = ?
	`, assignmentID)

	var matchLevel string
	var roleID int
	var matchStart time.Time

	if err := row.Scan(&matchLevel, &roleID, &matchStart); err != nil {
		cancel()
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		log.Printf("[ERROR]: DB error getting assignment info: %v", err)
		return types.ErrInternalServer
	}
	cancel()

	wageRow, cancelWage := db.queryRow(`
		SELECT id, fee
		FROM wages
		WHERE match_level = ? AND role_in_match = ? AND valid_from <= ?
		ORDER BY valid_from DESC
		LIMIT 1
	`, matchLevel, roleID, matchStart)

	var wagesID int
	var fee float64
	if err := wageRow.Scan(&wagesID, &fee); err != nil {
		cancelWage()
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("[ERROR]: No wage found for level %s role %d", matchLevel, roleID)
			return types.ErrNotFound
		}
		log.Printf("[ERROR]: DB error getting wage: %v", err)
		return types.ErrInternalServer
	}
	cancelWage()

	_, err := db.exec(`
		INSERT INTO payouts (assignment_id, wages_id, amount, status)
		VALUES (?, ?, ?, 'pending')
	`, assignmentID, wagesID, fee)

	if err != nil {
		log.Printf("[ERROR]: DB error inserting payout: %v", err)
		return types.ErrInternalServer
	}
	return nil
}

type PendingPayout struct {
	RefereeID int     `json:"referee_id"`
	Amount    float64 `json:"amount"`
}

func (db *Database) GetPendingPayouts(refereeIDs []int) ([]PendingPayout, types.ErrorApi) {
	if len(refereeIDs) == 0 {
		return []PendingPayout{}, nil
	}

	query := `
		SELECT ma.referee_id, SUM(p.amount) 
		FROM payouts p
		JOIN match_assignments ma ON p.assignment_id = ma.id
		WHERE p.status = 'pending' AND ma.referee_id IN (`
	
	args := make([]interface{}, len(refereeIDs))
	for i, id := range refereeIDs {
		args[i] = id
		if i > 0 {
			query += ", "
		}
		query += "?"
	}
	query += `) GROUP BY ma.referee_id`

	rows, cancel, err := db.query(query, args...)
	defer cancel()
	if err != nil {
		log.Printf("[ERROR]: DB error getting pending payouts: %v", err)
		return nil, types.ErrInternalServer
	}

	var results []PendingPayout
	for rows.Next() {
		var p PendingPayout
		if err := rows.Scan(&p.RefereeID, &p.Amount); err != nil {
			log.Printf("[ERROR]: DB error scanning pending payout: %v", err)
			return nil, types.ErrInternalServer
		}
		results = append(results, p)
	}
	rows.Close()

	return results, nil
}

func (db *Database) MarkPayoutsSent(refereeIDs []int) types.ErrorApi {
	if len(refereeIDs) == 0 {
		return nil
	}

	query := `
		UPDATE payouts p
		JOIN match_assignments ma ON p.assignment_id = ma.id
		SET p.status = 'sent'
		WHERE p.status = 'pending' AND ma.referee_id IN (`
	
	args := make([]interface{}, len(refereeIDs))
	for i, id := range refereeIDs {
		args[i] = id
		if i > 0 {
			query += ", "
		}
		query += "?"
	}
	query += `)`

	_, err := db.exec(query, args...)
	if err != nil {
		log.Printf("[ERROR]: DB error marking payouts sent: %v", err)
		return types.ErrInternalServer
	}

	return nil
}
