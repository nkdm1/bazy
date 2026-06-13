package database

import (
	"testing"
)

func TestGetMatchLevelID(t *testing.T) {
	db := testDB(t)

	_, cleanupLevel := createTestMatchLevel(t, db)
	defer cleanupLevel()

	t.Run("returns a valid ID for existing match level", func(t *testing.T) {
		// 'okregowa' may already exist in the DB from prior tests/data;
		// we just verify a valid positive ID is returned.
		id, err := db.GetMatchLevelID("okregowa")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if id <= 0 {
			t.Errorf("expected a positive match level ID, got %d", id)
		}
	})

	t.Run("returns ErrNotFound for unknown match level", func(t *testing.T) {
		_, err := db.GetMatchLevelID("nonexistent_level")
		if err == nil {
			t.Fatal("expected error for unknown match level, got nil")
		}
	})
}

func TestGetRoleInMatchID(t *testing.T) {
	db := testDB(t)

	_, cleanupRole := createTestRoleInMatch(t, db)
	defer cleanupRole()

	t.Run("returns a valid ID for existing role", func(t *testing.T) {
		// 'umpire' may already exist in the DB; we just verify a valid positive ID is returned.
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
	})
}

func TestInsertWage(t *testing.T) {
	db := testDB(t)

	levelID, cleanupLevel := createTestMatchLevel(t, db)
	defer cleanupLevel()

	roleID, cleanupRole := createTestRoleInMatch(t, db)
	defer cleanupRole()

	t.Run("successfully inserts a new wage row", func(t *testing.T) {
		err := db.InsertWage(levelID, roleID, 200.00)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		// Cleanup the inserted wage
		defer db.exec(`DELETE FROM wages WHERE match_level = ? AND role_in_match = ? AND fee = 200.00`, levelID, roleID)
	})
}
