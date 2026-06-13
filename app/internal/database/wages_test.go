package database

import (
	"errors"
	"testing"

	"github.com/nkdm1/bazy/internal/types"
)

func TestGetRoleInMatchID(t *testing.T) {
	db := testDB(t)

	_, cleanupRole := createTestRoleInMatch(t, db)
	defer cleanupRole()

	t.Run("returns a valid ID for existing role", func(t *testing.T) {
		id, err := db.GetRoleInMatchID("umpire")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if id <= 0 {
			t.Errorf("expected a positive role ID, got %d", id)
		}
	})

	t.Run("returns ErrNotFound for unknown role", func(t *testing.T) {
		_, err := db.GetRoleInMatchID("nonexistent_role")
		if err == nil {
			t.Fatal("expected error for unknown role, got nil")
		}
		if !errors.Is(err, types.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestInsertWage(t *testing.T) {
	db := testDB(t)

	roleID, cleanupRole := createTestRoleInMatch(t, db)
	defer cleanupRole()

	t.Run("successfully inserts a new wage row", func(t *testing.T) {
		err := db.InsertWage("okregowa", roleID, 200.00)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		// Cleanup the inserted wage
		defer db.exec(`DELETE FROM wages WHERE match_level = 'okregowa' AND role_in_match = ? AND fee = 200.00`, roleID)
	})
}
