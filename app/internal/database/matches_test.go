package database

import (
	"errors"
	"testing"

	"github.com/nkdm1/bazy/internal/types"
)

func TestGetMatchesForUpcomingWeek(t *testing.T) {
	db := testDB(t)

	// Create an upcoming match (in 2 days)
	matchUpcomingID, homeTeamID, awayTeamID, cleanupUpcoming := createTestMatch(t, db, "scheduled", 2)
	defer cleanupUpcoming()

	// Create a far match (in 10 days)
	_, _, _, cleanupFar := createTestMatch(t, db, "scheduled", 10)
	defer cleanupFar()

	matches, apiErr := db.GetMatchesForUpcomingWeek()

	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	// We only check if the matchUpcomingID is in the results
	// because there might be other matches in the database from manual testing.
	var found *Match
	for i := range matches {
		if matches[i].ID == matchUpcomingID {
			found = &matches[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected to find upcoming match %d in results, but didn't", matchUpcomingID)
	}

	if found.Status != "scheduled" {
		t.Errorf("expected status 'scheduled', got: %s", found.Status)
	}

	if found.HomeTeamID != homeTeamID || found.AwayTeamID != awayTeamID {
		t.Errorf("unexpected team IDs: home=%d away=%d",
			found.HomeTeamID, found.AwayTeamID)
	}
}

func TestMarkMatchAsCompleted(t *testing.T) {
	db := testDB(t)

	t.Run("successfully marks a scheduled match as completed", func(t *testing.T) {
		matchID, _, _, cleanup := createTestMatch(t, db, "scheduled", 2)
		defer cleanup()

		err := db.MarkMatchAsCompleted(matchID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("returns ErrNotFound for non-existent match ID", func(t *testing.T) {
		nonExistentID := 999999
		err := db.MarkMatchAsCompleted(nonExistentID)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}

		// Verifies it returns the correct API error definition
		if !errors.Is(err, types.ErrNotFound) {
			t.Errorf("expected a not found error configuration, got: %v", err)
		}
	})
}
