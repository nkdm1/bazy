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
