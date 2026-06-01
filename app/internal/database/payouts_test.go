package database

import (
	"testing"
	"time"
)

func TestGetMonthlyPayoutBudget(t *testing.T) {
	db := testDB(t)

	// Seed wstawił jedną wypłatę 150.00 zapłaconą w czerwcu 2026

	t.Run("returns correct sum for june 2026", func(t *testing.T) {
		budget, apiErr := db.GetMonthlyPayoutBudget(2026, time.June)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		// Seed wstawił dokładnie 150.00
		if budget != testPayout.Amount {
			t.Errorf("expected 150.00, got: %.2f", budget)
		}
	})

	t.Run("returns 0 for month with no payouts", func(t *testing.T) {
		budget, apiErr := db.GetMonthlyPayoutBudget(2026, time.January)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		// Brak wypłat w styczniu → COALESCE zwraca 0
		if budget != 0 {
			t.Errorf("expected 0.00, got: %.2f", budget)
		}
	})
}

func TestGetTotalEarningsByRefereeID(t *testing.T) {
	db := testDB(t)

	t.Run("returns correct total earnings for an active referee with paid assignments", func(t *testing.T) {
		earnings, apiErr := db.GetTotalEarningsByRefereeID(testReferee.ID)
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		// Checks against the seeded testPayout value (150.00)
		if earnings != testPayout.Amount {
			t.Errorf("expected earnings to be %.2f, got: %.2f", testPayout.Amount, earnings)
		}
	})

	t.Run("returns 0 for a referee ID that has no history records", func(t *testing.T) {
		nonExistentRefereeID := 8888
		earnings, apiErr := db.GetTotalEarningsByRefereeID(nonExistentRefereeID)
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}

		// SQL COALESCE ensures it safely drops back down to 0.00 rather than failing
		if earnings != 0 {
			t.Errorf("expected earnings to be 0.00, got: %.2f", earnings)
		}
	})
}
