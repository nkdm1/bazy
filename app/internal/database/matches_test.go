package database

import (
	"errors"
	"testing"
	"time"

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

func TestCreateMatch(t *testing.T) {
	db := testDB(t)

	homeTeamID, cleanupHome := createTestTeam(t, db)
	defer cleanupHome()
	awayTeamID, cleanupAway := createTestTeam(t, db)
	defer cleanupAway()
	venueID, cleanupVenue := createTestVenue(t, db)
	defer cleanupVenue()

	t.Run("successfully inserts a match", func(t *testing.T) {
		start := time.Now().Add(24 * time.Hour)
		end := start.Add(2 * time.Hour)

		// Get names of teams and venue to verify name lookup
		var homeName, awayName, gymName string
		rowH, cancelH := db.queryRow(`SELECT name FROM teams WHERE id = ?`, homeTeamID)
		rowH.Scan(&homeName)
		cancelH()

		rowA, cancelA := db.queryRow(`SELECT name FROM teams WHERE id = ?`, awayTeamID)
		rowA.Scan(&awayName)
		cancelA()

		rowV, cancelV := db.queryRow(`SELECT gym_name FROM venues WHERE id = ?`, venueID)
		rowV.Scan(&gymName)
		cancelV()

		hID, err := db.GetTeamIDByName(homeName)
		if err != nil || hID != homeTeamID {
			t.Fatalf("failed GetTeamIDByName: %v, got %d, expected %d", err, hID, homeTeamID)
		}

		aID, err := db.GetTeamIDByName(awayName)
		if err != nil || aID != awayTeamID {
			t.Fatalf("failed GetTeamIDByName: %v, got %d, expected %d", err, aID, awayTeamID)
		}

		vID, err := db.GetVenueIDByName(gymName)
		if err != nil || vID <= 0 {
			t.Fatalf("failed GetVenueIDByName: %v, got %d, expected positive", err, vID)
		}

		err = db.CreateMatch(homeTeamID, awayTeamID, venueID, "okregowa", start, end)
		if err != nil {
			t.Fatalf("failed CreateMatch: %v", err)
		}

		// Cleanup inserted match
		defer func() {
			db.exec(`DELETE FROM matches WHERE home_team_id = ? AND away_team_id = ?`, homeTeamID, awayTeamID)
		}()
	})
}

