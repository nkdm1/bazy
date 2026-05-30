package database

import (
	"testing"
	"time"
)

func TestCheckRefereeAvailability(t *testing.T) {
	db := testDB(t)

	// Seed wstawił dostępność sędziego 1 na 2026-06-20
	availableDate := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	unavailableDate := time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)

	t.Run("referee is available on given date", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(1, availableDate)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if !available {
			t.Error("expected referee to be available, got false")
		}
	})

	t.Run("referee is not available on different date", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(1, unavailableDate)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if available {
			t.Error("expected referee to be unavailable, got true")
		}
	})

	t.Run("non-existent referee returns false", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(999, availableDate)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if available {
			t.Error("expected false for non-existent referee, got true")
		}
	})
}
