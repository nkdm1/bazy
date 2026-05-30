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
	if matches[0].ID != 1 {
		t.Errorf("expected match ID 1, got: %d", matches[0].ID)
	}

	if matches[0].Status != "scheduled" {
		t.Errorf("expected status 'scheduled', got: %s", matches[0].Status)
	}

	if matches[0].HomeTeamID != 1 || matches[0].AwayTeamID != 2 {
		t.Errorf("unexpected team IDs: home=%d away=%d",
			matches[0].HomeTeamID, matches[0].AwayTeamID)
	}
}
