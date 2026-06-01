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
