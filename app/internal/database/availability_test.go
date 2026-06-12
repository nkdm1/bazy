package database

import (
	"testing"
	"time"
)

func TestCheckRefereeAvailability(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	date := time.Date(2027, 10, 10, 0, 0, 0, 0, time.UTC)

	// Add availability manually for testing Check
	err := db.AddRefereeAvailability(refereeID, date)
	if err != nil {
		t.Fatalf("failed to add availability: %v", err)
	}
	defer db.RemoveRefereeAvailability(refereeID, date)

	t.Run("referee is available on given date", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(refereeID, date)
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if !available {
			t.Error("expected referee to be available, got false")
		}
	})

	t.Run("referee is not available on different date", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(refereeID, date.Add(time.Hour*24))
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if available {
			t.Error("expected referee to be unavailable, got true")
		}
	})

	t.Run("non-existent referee returns false", func(t *testing.T) {
		available, apiErr := db.CheckRefereeAvailability(999999, date)
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		if available {
			t.Error("expected false for non-existent referee, got true")
		}
	})
}

func TestAddAndRemoveRefereeAvailability(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupRef := createTestReferee(t, db)
	defer cleanupRef()

	dateToAdd := time.Date(2027, 11, 11, 0, 0, 0, 0, time.UTC)

	t.Run("add availability successfully", func(t *testing.T) {
		err := db.AddRefereeAvailability(refereeID, dateToAdd)
		if err != nil {
			t.Fatalf("expected no error adding availability, got: %v", err)
		}

		available, apiErr := db.CheckRefereeAvailability(refereeID, dateToAdd)
		if apiErr != nil {
			t.Fatalf("expected no error checking availability, got: %v", apiErr)
		}
		if !available {
			t.Error("expected referee to be available after adding")
		}
	})

	t.Run("remove availability successfully", func(t *testing.T) {
		err := db.RemoveRefereeAvailability(refereeID, dateToAdd)
		if err != nil {
			t.Fatalf("expected no error removing availability, got: %v", err)
		}

		available, apiErr := db.CheckRefereeAvailability(refereeID, dateToAdd)
		if apiErr != nil {
			t.Fatalf("expected no error checking availability, got: %v", apiErr)
		}
		if available {
			t.Error("expected referee to be unavailable after removing")
		}
	})
}
