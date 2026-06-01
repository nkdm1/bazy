package database

import (
	"testing"
	"time"
)

func TestCheckRefereeAvailability(t *testing.T) {
	db := testDB(t)

	t.Run("referee is available on given date", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(
			testReferee.ID,
			testAvailability.AvailableDate,
		)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if !available {
			t.Error("expected referee to be available, got false")
		}
	})

	t.Run("referee is not available on different date", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(
			testReferee.ID,
			testAvailability.AvailableDate.Add(time.Hour*24),
		)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if available {
			t.Error("expected referee to be unavailable, got true")
		}
	})

	t.Run("non-existent referee returns false", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(
			testReferee.ID+1000,
			testAvailability.AvailableDate,
		)

		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if available {
			t.Error("expected false for non-existent referee, got true")
		}
	})
}
