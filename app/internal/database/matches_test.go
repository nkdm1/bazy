package database

import (
	"testing"
)

func TestGetMatchesForUpcomingWeek(t *testing.T) {
	db := testDB(t)

	matches, apiErr := db.GetMatchesForUpcomingWeek()

	// Nie powinno być błędu
	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	// Seed wstawił jeden mecz za 2 dni i jeden za 10 dni.
	// Funkcja powinna zwrócić TYLKO ten za 2 dni.
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got: %d", len(matches))
	}

	// Sprawdzamy czy to właściwy mecz
	if matches[0].ID != testMatchUpcoming.ID {
		t.Errorf("expected match ID %d, got: %d", testMatchUpcoming.ID, matches[0].ID)
	}

	if matches[0].Status != "scheduled" {
		t.Errorf("expected status 'scheduled', got: %s", matches[0].Status)
	}

	if matches[0].HomeTeamID != testTeamA.ID || matches[0].AwayTeamID != testTeamB.ID {
		t.Errorf("unexpected team IDs: home=%d away=%d",
			matches[0].HomeTeamID, matches[0].AwayTeamID)
	}
}
