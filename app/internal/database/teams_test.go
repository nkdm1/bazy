package database

import (
	"errors"
	"testing"

	"github.com/nkdm1/bazy/internal/types"
)

func TestCreateTeam(t *testing.T) {
	db := testDB(t)

	t.Run("successfully creates a new team", func(t *testing.T) {
		name := "Unique Team A"
		city := "City A"
		
		err := db.CreateTeam(name, city)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		// Cleanup inserted team to avoid conflicts in subsequent runs/tests
		defer func() {
			db.exec(`DELETE FROM teams WHERE name = ? AND city = ?`, name, city)
		}()
	})

	t.Run("returns ErrConflict if team name and city already exist", func(t *testing.T) {
		teamID, cleanupTeam := createTestTeam(t, db)
		defer cleanupTeam()

		// Get the team details we just created
		var name, city string
		row, cancel := db.queryRow(`SELECT name, city FROM teams WHERE id = ?`, teamID)
		if err := row.Scan(&name, &city); err != nil {
			cancel()
			t.Fatalf("failed to retrieve test team: %v", err)
		}
		cancel()

		// Try to create the same team again
		err := db.CreateTeam(name, city)
		if err == nil {
			t.Fatal("expected conflict error, got nil")
		}
		if !errors.Is(err, types.ErrConflict) {
			t.Errorf("expected ErrConflict, got: %v", err)
		}
	})
}
