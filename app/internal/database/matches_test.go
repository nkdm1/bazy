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

func TestGetUpcomingMatchesWithDetails(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupUpcoming := createTestMatch(t, db, "scheduled", 2)
	defer cleanupUpcoming()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssignment := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssignment()

	matches, apiErr := db.GetUpcomingMatchesWithDetails()
	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	var found *UpcomingMatch
	for i := range matches {
		if matches[i].ID == matchID {
			found = &matches[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected to find upcoming match in results")
	}

	if len(found.Assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(found.Assignments))
	}
	
	if found.Assignments[0].Role != "umpire" {
		t.Errorf("expected role umpire, got %s", found.Assignments[0].Role)
	}
}

func TestGetCompletedMatches(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupCompleted := createTestMatch(t, db, "completed", -2)
	defer cleanupCompleted()

	db.exec(`UPDATE matches SET home_team_points = 80, away_team_points = 70 WHERE id = ?`, matchID)

	matches, apiErr := db.GetCompletedMatches()
	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	var found *CompletedMatch
	for i := range matches {
		if matches[i].ID == matchID {
			found = &matches[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("expected to find completed match in results")
	}

	if found.HomeTeamPoints == nil || *found.HomeTeamPoints != 80 {
		t.Errorf("expected home team points 80, got %v", found.HomeTeamPoints)
	}

	if found.AwayTeamPoints == nil || *found.AwayTeamPoints != 70 {
		t.Errorf("expected away team points 70, got %v", found.AwayTeamPoints)
	}
}

func TestGetMatchDetails(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssignment := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssignment()

	db.exec(`UPDATE matches SET home_team_points = 80, away_team_points = 70 WHERE id = ?`, matchID)

	matchDetails, apiErr := db.GetMatchDetails(matchID)
	if apiErr != nil {
		t.Fatalf("expected no error, got: %v", apiErr)
	}

	if matchDetails.ID != matchID {
		t.Errorf("expected match ID %d, got %d", matchID, matchDetails.ID)
	}

	if len(matchDetails.Assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(matchDetails.Assignments))
	}
	
	if matchDetails.Assignments[0].Role != "umpire" {
		t.Errorf("expected role umpire, got %s", matchDetails.Assignments[0].Role)
	}

	if matchDetails.HomeTeamPoints == nil || *matchDetails.HomeTeamPoints != 80 {
		t.Errorf("expected home team points 80, got %v", matchDetails.HomeTeamPoints)
	}

	if matchDetails.AwayTeamPoints == nil || *matchDetails.AwayTeamPoints != 70 {
		t.Errorf("expected away team points 70, got %v", matchDetails.AwayTeamPoints)
	}
}

func TestCancelMatch(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	assignID, cleanupAssignment := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssignment()

	err := db.CancelMatch(matchID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var status string
	rowStatus, cancelStatus := db.queryRow(`SELECT status FROM matches WHERE id = ?`, matchID)
	rowStatus.Scan(&status)
	cancelStatus()
	if status != "cancelled" {
		t.Errorf("expected match status 'cancelled', got %s", status)
	}

	var assignStatus string
	rowAssign, cancelAssign := db.queryRow(`SELECT assignment_status FROM match_assignments WHERE id = ?`, assignID)
	rowAssign.Scan(&assignStatus)
	cancelAssign()
	if assignStatus != "cancelled" {
		t.Errorf("expected assignment status 'cancelled', got %s", assignStatus)
	}

	errNotFound := db.CancelMatch(999999)
	if !errors.Is(errNotFound, types.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", errNotFound)
	}
}

func TestRescheduleMatch(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	start := time.Now().Add(48 * time.Hour).Truncate(time.Second)
	end := start.Add(2 * time.Hour)

	err := db.RescheduleMatch(matchID, start, end)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var newStart, newEnd time.Time
	row, cancel := db.queryRow(`SELECT match_start, match_end FROM matches WHERE id = ?`, matchID)
	row.Scan(&newStart, &newEnd)
	cancel()

	if !newStart.Equal(start) {
		t.Errorf("expected match_start %v, got %v", start, newStart)
	}
	if !newEnd.Equal(end) {
		t.Errorf("expected match_end %v, got %v", end, newEnd)
	}

	errNotFound := db.RescheduleMatch(999999, start, end)
	if !errors.Is(errNotFound, types.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", errNotFound)
	}
}

func TestAssignReferee(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupRole := createTestRoleInMatch(t, db)
	defer cleanupRole()

	err := db.AssignReferee(matchID, refereeID, "umpire")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var status string
	row, cancel := db.queryRow(`SELECT assignment_status FROM match_assignments WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)
	row.Scan(&status)
	cancel()

	if status != "pending" {
		t.Errorf("expected status 'pending', got %s", status)
	}

	errConflict := db.AssignReferee(matchID, refereeID, "umpire")
	if !errors.Is(errConflict, types.ErrConflict) {
		t.Errorf("expected ErrConflict, got: %v", errConflict)
	}
}

func TestRevokeAssignment(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	err := db.RevokeAssignment(matchID, refereeID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var status string
	row, cancel := db.queryRow(`SELECT assignment_status FROM match_assignments WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)
	row.Scan(&status)
	cancel()

	if status != "cancelled" {
		t.Errorf("expected status 'cancelled', got %s", status)
	}

	errNotFound := db.RevokeAssignment(999, 999)
	if !errors.Is(errNotFound, types.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", errNotFound)
	}
}

func TestRespondToAssignment(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	// Update assignment to pending first (createTestMatchAssignment sets it to accepted by default or pending? Let's assume accepted if we don't know, we will just update it)
	db.exec(`UPDATE match_assignments SET assignment_status = 'pending' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)

	err := db.RespondToAssignment(matchID, refereeID, true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var status string
	row, cancel := db.queryRow(`SELECT assignment_status FROM match_assignments WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)
	row.Scan(&status)
	cancel()

	if status != "accepted" {
		t.Errorf("expected status 'accepted', got %s", status)
	}

	errNotFound := db.RespondToAssignment(999, 999, true)
	if !errors.Is(errNotFound, types.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", errNotFound)
	}
}

func TestGetPendingAssignments(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	// Ensure it's pending
	db.exec(`UPDATE match_assignments SET assignment_status = 'pending' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)

	pending, err := db.GetPendingAssignments(refereeID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	found := false
	for _, p := range pending {
		if p.MatchID == matchID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find match %d in pending assignments", matchID)
	}
}

func TestCancelAcceptedAssignment(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	db.exec(`UPDATE match_assignments SET assignment_status = 'pending' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)

	errNotFound := db.CancelAcceptedAssignment(matchID, refereeID)
	if !errors.Is(errNotFound, types.ErrNotFound) {
		t.Errorf("expected ErrNotFound for pending assignment, got: %v", errNotFound)
	}

	db.exec(`UPDATE match_assignments SET assignment_status = 'accepted' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)

	err := db.CancelAcceptedAssignment(matchID, refereeID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var status string
	row, cancel := db.queryRow(`SELECT assignment_status FROM match_assignments WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)
	row.Scan(&status)
	cancel()

	if status != "cancelled" {
		t.Errorf("expected status 'cancelled', got %s", status)
	}
}

func TestGetAcceptedAssignments(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	db.exec(`UPDATE match_assignments SET assignment_status = 'accepted' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)

	accepted, err := db.GetAcceptedAssignments(refereeID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	found := false
	for _, a := range accepted {
		if a.MatchID == matchID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find match %d in accepted assignments", matchID)
	}
}

func TestMarkNoShow(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	db.exec(`UPDATE match_assignments SET assignment_status = 'pending' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)

	errNotFound := db.MarkNoShow(matchID, refereeID)
	if !errors.Is(errNotFound, types.ErrNotFound) {
		t.Errorf("expected ErrNotFound for pending assignment, got: %v", errNotFound)
	}

	db.exec(`UPDATE match_assignments SET assignment_status = 'accepted' WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)

	err := db.MarkNoShow(matchID, refereeID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var status string
	row, cancel := db.queryRow(`SELECT assignment_status FROM match_assignments WHERE match_id = ? AND referee_id = ?`, matchID, refereeID)
	row.Scan(&status)
	cancel()

	if status != "noshow" {
		t.Errorf("expected status 'noshow', got %s", status)
	}
}

func TestGetMatchAssignmentHistory(t *testing.T) {
	db := testDB(t)

	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	_, cleanupAssign := createTestMatchAssignment(t, db, refereeID, matchID)
	defer cleanupAssign()

	history, err := db.GetMatchAssignmentHistory(matchID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 history record, got %d", len(history))
	} else {
		if history[0].RefereeID != refereeID {
			t.Errorf("expected referee %d, got %d", refereeID, history[0].RefereeID)
		}
	}
}

func TestSubmitMatchScore(t *testing.T) {
	db := testDB(t)

	// Create match
	matchID, _, _, cleanupMatch := createTestMatch(t, db, "scheduled", 2)
	defer cleanupMatch()

	// Ensure crew_chief role exists
	var roleID int
	roleRow, roleCancel := db.queryRow(`SELECT id FROM role_in_match WHERE match_role = 'crew_chief'`)
	err := roleRow.Scan(&roleID)
	roleCancel()
	if err != nil {
		res, _ := db.exec(`INSERT INTO role_in_match (match_role) VALUES ('crew_chief')`)
		id, _ := res.LastInsertId()
		roleID = int(id)
	}

	// Create wage for this match
	db.exec(`INSERT INTO wages (match_level, role_in_match, fee, valid_from) VALUES ('okregowa', ?, 150.00, '2020-01-01')`, roleID)

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	db.exec(`INSERT INTO match_assignments (referee_id, match_id, role, assignment_status) VALUES (?, ?, ?, 'accepted')`, refereeID, matchID, roleID)

	// Test non-crew-chief or pending assignment
	errForbidden := db.SubmitMatchScore(matchID, refereeID+1, 80, 70)
	if !errors.Is(errForbidden, types.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got: %v", errForbidden)
	}

	err = db.SubmitMatchScore(matchID, refereeID, 80, 70)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var status string
	var hPts, aPts int
	matchRow, matchCancel := db.queryRow(`SELECT status, home_team_points, away_team_points FROM matches WHERE id = ?`, matchID)
	matchRow.Scan(&status, &hPts, &aPts)
	matchCancel()
	if status != "completed" || hPts != 80 || aPts != 70 {
		t.Errorf("match not updated correctly, got: %s %d %d", status, hPts, aPts)
	}

	// Verify payout was created
	var count int
	countRow, countCancel := db.queryRow(`SELECT count(*) FROM payouts p JOIN match_assignments ma ON p.assignment_id = ma.id WHERE ma.match_id = ?`, matchID)
	countRow.Scan(&count)
	countCancel()
	if count != 1 {
		t.Errorf("expected 1 payout, got %d", count)
	}
}
