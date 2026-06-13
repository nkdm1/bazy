package database

import (
	"testing"
)

func TestCreateVenue(t *testing.T) {
	db := testDB(t)

	t.Run("successfully creates a new venue with full address", func(t *testing.T) {
		gymName := "Madison Square Garden"
		postcode := "10-001"
		city := "New York"
		street := "Pennsylvania Ave"
		streetNumber := "4"
		flatNumber := "Suite 1"

		err := db.CreateVenue(gymName, postcode, city, street, streetNumber, flatNumber)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Cleanup
		defer func() {
			db.exec(`
				DELETE FROM venues WHERE gym_name = ?;
			`, gymName)
			db.exec(`
				DELETE FROM address WHERE postcode = ? AND city = ? AND street = ? AND street_number = ? AND flat_number = ?;
			`, postcode, city, street, streetNumber, flatNumber)
		}()
	})
}
