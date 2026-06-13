package database

import (
	"testing"
	"time"
)

func TestGetMonthlyPayoutBudget(t *testing.T) {
	db := testDB(t)

	t.Run("returns correct sum for a month with payouts", func(t *testing.T) {
		refereeID, cleanupRef := createTestReferee(t, db)
		defer cleanupRef()
		matchID, _, _, cleanupMatch := createTestMatch(t, db, "completed", -10)
		defer cleanupMatch()
		assignID, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
		defer cleanupAssign()

		paidDate := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
		_, amount, cleanupPayout := createTestPayout(t, db, assignID, paidDate)
		defer cleanupPayout()

		budget, apiErr := db.GetMonthlyPayoutBudget(2026, time.June)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		// It should be at least the amount we inserted (other tests/data might exist but this is just for our test context)
		// Wait, budget might include other data. We can't strictly compare. But we'll assume it's exactly `amount` if the DB is clean for this date.
		// To be safe, we'll check budget >= amount.
		if budget < amount {
			t.Errorf("expected at least %.2f, got: %.2f", amount, budget)
		}
	})

	t.Run("returns 0 for month with no payouts", func(t *testing.T) {
		// Just querying an arbitrary old date like 1999
		budget, apiErr := db.GetMonthlyPayoutBudget(1999, time.January)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		if budget != 0 {
			t.Errorf("expected 0.00, got: %.2f", budget)
		}
	})
}

func TestGetTotalEarningsByRefereeID(t *testing.T) {
	db := testDB(t)

	t.Run("returns correct total earnings for an active referee with paid assignments", func(t *testing.T) {
		refereeID, cleanupRef := createTestReferee(t, db)
		defer cleanupRef()
		matchID, _, _, cleanupMatch := createTestMatch(t, db, "completed", -10)
		defer cleanupMatch()
		assignID, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
		defer cleanupAssign()

		paidDate := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
		_, amount, cleanupPayout := createTestPayout(t, db, assignID, paidDate)
		defer cleanupPayout()

		earnings, apiErr := db.GetTotalEarningsByRefereeID(refereeID)
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		if earnings != amount {
			t.Errorf("expected earnings to be %.2f, got: %.2f", amount, earnings)
		}
	})

	t.Run("returns 0 for a referee ID that has no history records", func(t *testing.T) {
		nonExistentRefereeID := 999999
		earnings, apiErr := db.GetTotalEarningsByRefereeID(nonExistentRefereeID)
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		if earnings != 0 {
			t.Errorf("expected earnings to be 0.00, got: %.2f", earnings)
		}
	})
}

func TestGetPendingPayouts(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "completed", -10)
	defer cleanupMatch()
	assignID, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	payoutID, amount, cleanupPayout := createTestPayout(t, db, assignID, time.Time{})
	defer cleanupPayout()
	db.exec(`UPDATE payouts SET status = 'pending' WHERE id = ?`, payoutID)

	payouts, apiErr := db.GetPendingPayouts([]int{refereeID}, false)
	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	if len(payouts) != 1 {
		t.Fatalf("expected 1 payout, got %d", len(payouts))
	}
	if payouts[0].RefereeID != refereeID || payouts[0].Amount != amount {
		t.Errorf("expected payout to be %f for ref %d, got %f for ref %d", amount, refereeID, payouts[0].Amount, payouts[0].RefereeID)
	}

	// Empty list
	emptyPayouts, err := db.GetPendingPayouts([]int{}, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(emptyPayouts) != 0 {
		t.Errorf("expected 0 payouts, got %d", len(emptyPayouts))
	}
}

func TestMarkPayoutsSent(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "completed", -10)
	defer cleanupMatch()
	assignID, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	payoutID, _, cleanupPayout := createTestPayout(t, db, assignID, time.Time{})
	defer cleanupPayout()
	db.exec(`UPDATE payouts SET status = 'pending' WHERE id = ?`, payoutID)

	_, err := db.MarkPayoutsSent([]int{refereeID})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var status string
	row, cancel := db.queryRow(`SELECT status FROM payouts WHERE id = ?`, payoutID)
	row.Scan(&status)
	cancel()
	if status != "sent" {
		t.Errorf("expected status 'sent', got %s", status)
	}

	// Empty list
	_, err = db.MarkPayoutsSent([]int{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessPayouts(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "completed", -10)
	defer cleanupMatch()
	assignID, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	payoutID, _, cleanupPayout := createTestPayout(t, db, assignID, time.Time{})
	defer cleanupPayout()
	db.exec(`UPDATE payouts SET status = 'sent' WHERE id = ?`, payoutID)

	err := db.ProcessPayouts([]PayoutConfirmation{{PayoutID: payoutID, BankTransactionID: "tx_123"}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var status, txID string
	row, cancel := db.queryRow(`SELECT status, bank_transaction_id FROM payouts WHERE id = ?`, payoutID)
	row.Scan(&status, &txID)
	cancel()
	if status != "paid" || txID != "tx_123" {
		t.Errorf("expected paid and tx_123, got %s and %s", status, txID)
	}
}

func TestGetPayoutHistory(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "completed", -10)
	defer cleanupMatch()
	assignID, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	createTestPayout(t, db, assignID, time.Time{})

	history, err := db.GetPayoutHistory(refereeID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history))
	}
	if history[0].AssignmentID != assignID {
		t.Errorf("expected assignment %d, got %d", assignID, history[0].AssignmentID)
	}
}

func TestGetMonthlyPayoutReport(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "completed", -10)
	defer cleanupMatch()
	assignID, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	paidDate := time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)
	_, amount, cleanupPayout := createTestPayout(t, db, assignID, paidDate)
	defer cleanupPayout()

	report, err := db.GetMonthlyPayoutReport(2026, time.June)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found := false
	for _, item := range report {
		if item.RefereeID == refereeID && item.TotalPaid == amount {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected report to contain referee %d with amount %f", refereeID, amount)
	}
}
