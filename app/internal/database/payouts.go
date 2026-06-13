package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
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

func (db *Database) GetPendingPayouts(refereeIDs []int, all bool) ([]PendingPayout, types.ErrorApi) {
	if !all && len(refereeIDs) == 0 {
		return []PendingPayout{}, nil
	}

	query := `
		SELECT ma.referee_id, SUM(p.amount) 
		FROM payouts p
		JOIN match_assignments ma ON p.assignment_id = ma.id
		WHERE p.status = 'pending'`
	
	var args []interface{}
	if !all {
		query += ` AND ma.referee_id IN (`
		args = make([]interface{}, len(refereeIDs))
		for i, id := range refereeIDs {
			args[i] = id
			if i > 0 {
				query += ", "
			}
			query += "?"
		}
		query += `)`
	}
	query += ` GROUP BY ma.referee_id`

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

type SentPayoutResult struct {
	PayoutID          int    `json:"payout_id"`
	BankTransactionID string `json:"bank_transaction_id"`
}

func (db *Database) MarkPayoutsSent(refereeIDs []int) ([]SentPayoutResult, types.ErrorApi) {
	if len(refereeIDs) == 0 {
		return []SentPayoutResult{}, nil
	}

	query := `
		SELECT p.id
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
	query += `)`

	rows, cancel, err := db.query(query, args...)
	defer cancel()
	if err != nil {
		log.Printf("[ERROR]: DB error getting payouts to mark sent: %v", err)
		return nil, types.ErrInternalServer
	}

	var payoutIDs []int
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err == nil {
			payoutIDs = append(payoutIDs, pid)
		}
	}
	rows.Close()

	if len(payoutIDs) == 0 {
		return []SentPayoutResult{}, nil
	}

	results := []SentPayoutResult{}
	tx, err := db.instance.Begin()
	if err != nil {
		log.Printf("[ERROR]: Begin tx error: %v", err)
		return nil, types.ErrInternalServer
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE payouts SET status = 'sent', bank_transaction_id = ? WHERE id = ?`)
	if err != nil {
		return nil, types.ErrInternalServer
	}
	defer stmt.Close()

	for _, pid := range payoutIDs {
		txID := fmt.Sprintf("TX%d", rand.Intn(1000000)+100000)
		_, err := stmt.Exec(txID, pid)
		if err != nil {
			log.Printf("[ERROR]: Exec error: %v", err)
			return nil, types.ErrInternalServer
		}
		results = append(results, SentPayoutResult{PayoutID: pid, BankTransactionID: txID})
	}

	if err := tx.Commit(); err != nil {
		return nil, types.ErrInternalServer
	}

	return results, nil
}

type PayoutConfirmation struct {
	PayoutID          int    `json:"payout_id"`
	BankTransactionID string `json:"bank_transaction_id"`
}

func (db *Database) ProcessPayouts(confirmations []PayoutConfirmation) types.ErrorApi {
	if len(confirmations) == 0 {
		return nil
	}

	tx, err := db.instance.BeginTx(context.Background(), nil)
	if err != nil {
		log.Printf("[ERROR]: DB error starting transaction: %v", err)
		return types.ErrInternalServer
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		UPDATE payouts 
		SET status = 'paid', bank_transaction_id = ?, paid_at = NOW() 
		WHERE id = ? AND status = 'sent'
	`)
	if err != nil {
		log.Printf("[ERROR]: DB error preparing statement: %v", err)
		return types.ErrInternalServer
	}
	defer stmt.Close()

	for _, conf := range confirmations {
		_, err := stmt.Exec(conf.BankTransactionID, conf.PayoutID)
		if err != nil {
			log.Printf("[ERROR]: DB error processing payout %d: %v", conf.PayoutID, err)
			return types.ErrInternalServer
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR]: DB error committing processing payouts: %v", err)
		return types.ErrInternalServer
	}

	return nil
}

type PayoutHistoryItem struct {
	ID                int     `json:"id"`
	AssignmentID      int     `json:"assignment_id"`
	Amount            float64 `json:"amount"`
	Status            string  `json:"status"`
	BankTransactionID *string `json:"bank_transaction_id"`
	PaidAt            *string `json:"paid_at"`
}

func (db *Database) GetPayoutHistory(refereeID int) ([]PayoutHistoryItem, types.ErrorApi) {
	rows, cancel, err := db.query(`
		SELECT p.id, p.assignment_id, p.amount, p.status, p.bank_transaction_id, p.paid_at
		FROM payouts p
		JOIN match_assignments ma ON p.assignment_id = ma.id
		WHERE ma.referee_id = ?
		ORDER BY p.id DESC
	`, refereeID)
	defer cancel()
	if err != nil {
		log.Printf("[ERROR]: DB error fetching payout history: %v", err)
		return nil, types.ErrInternalServer
	}

	list := make([]PayoutHistoryItem, 0)
	for rows.Next() {
		var item PayoutHistoryItem
		if err := rows.Scan(&item.ID, &item.AssignmentID, &item.Amount, &item.Status, &item.BankTransactionID, &item.PaidAt); err != nil {
			log.Printf("[ERROR]: DB error scanning payout history item: %v", err)
			return nil, types.ErrInternalServer
		}
		list = append(list, item)
	}
	rows.Close()

	return list, nil
}

type PayoutReportItem struct {
	RefereeID int     `json:"referee_id"`
	TotalPaid float64 `json:"total_paid"`
}

func (db *Database) GetMonthlyPayoutReport(year int, month time.Month) ([]PayoutReportItem, types.ErrorApi) {
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Second)

	rows, cancel, err := db.query(`
		SELECT ma.referee_id, SUM(p.amount)
		FROM payouts p
		JOIN match_assignments ma ON p.assignment_id = ma.id
		WHERE p.status = 'paid'
			AND p.paid_at BETWEEN ? AND ?
		GROUP BY ma.referee_id
		ORDER BY ma.referee_id ASC
	`, firstDay, lastDay)
	defer cancel()

	if err != nil {
		log.Printf("[ERROR]: DB error fetching monthly payout report: %v", err)
		return nil, types.ErrInternalServer
	}

	list := make([]PayoutReportItem, 0)
	for rows.Next() {
		var item PayoutReportItem
		if err := rows.Scan(&item.RefereeID, &item.TotalPaid); err != nil {
			log.Printf("[ERROR]: DB error scanning report item: %v", err)
			return nil, types.ErrInternalServer
		}
		list = append(list, item)
	}
	rows.Close()

	return list, nil
}
