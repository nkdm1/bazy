package database

import (
	"fmt"
	"testing"
	"time"
)

func createTestUser(t *testing.T, db *Database) (int, func()) {
	t.Helper()
	email := fmt.Sprintf("testuser_%d@test.com", time.Now().UnixNano())
	res, err := db.exec(`INSERT INTO users (email, password_hash, name, surname, role) VALUES (?, 'hash', 'Test', 'User', 'referee')`, email)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id), func() {
		db.exec(`DELETE FROM users WHERE id = ?`, id)
	}
}

func createTestAddress(t *testing.T, db *Database) (int, func()) {
	t.Helper()
	res, err := db.exec(`INSERT INTO address (postcode, city, street, street_number) VALUES ('00-000', 'City', 'Street', '1')`)
	if err != nil {
		t.Fatalf("failed to create test address: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id), func() {
		db.exec(`DELETE FROM address WHERE id = ?`, id)
	}
}

func createTestReferee(t *testing.T, db *Database) (int, func()) {
	t.Helper()
	userID, cleanupUser := createTestUser(t, db)
	addressID, cleanupAddress := createTestAddress(t, db)

	res, err := db.exec(`INSERT INTO referees (user_id, address_id) VALUES (?, ?)`, userID, addressID)
	if err != nil {
		t.Fatalf("failed to create test referee: %v", err)
	}
	id, _ := res.LastInsertId()

	return int(id), func() {
		db.exec(`DELETE FROM referees WHERE id = ?`, id)
		cleanupAddress()
		cleanupUser()
	}
}

func createTestTeam(t *testing.T, db *Database) (int, func()) {
	t.Helper()
	name := fmt.Sprintf("Team %d", time.Now().UnixNano())
	res, err := db.exec(`INSERT INTO teams (name, city) VALUES (?, 'City')`, name)
	if err != nil {
		t.Fatalf("failed to create test team: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id), func() {
		db.exec(`DELETE FROM teams WHERE id = ?`, id)
	}
}

func createTestVenue(t *testing.T, db *Database) (int, func()) {
	t.Helper()
	addressID, cleanupAddress := createTestAddress(t, db)
	res, err := db.exec(`INSERT INTO venues (gym_name, address_id) VALUES ('Gym', ?)`, addressID)
	if err != nil {
		t.Fatalf("failed to create test venue: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id), func() {
		db.exec(`DELETE FROM venues WHERE id = ?`, id)
		cleanupAddress()
	}
}



func createTestRoleInMatch(t *testing.T, db *Database) (int, func()) {
	t.Helper()
	res, err := db.exec(`INSERT INTO role_in_match (match_role) VALUES ('umpire')`)
	if err != nil {
		t.Fatalf("failed to create test role in match: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id), func() {
		db.exec(`DELETE FROM role_in_match WHERE id = ?`, id)
	}
}

func createTestMatch(t *testing.T, db *Database, status string, daysFromNow int) (int, int, int, func()) {
	t.Helper()
	homeTeamID, cleanupHome := createTestTeam(t, db)
	awayTeamID, cleanupAway := createTestTeam(t, db)
	venueID, cleanupVenue := createTestVenue(t, db)

	query := fmt.Sprintf(`INSERT INTO matches (match_start, match_end, level_of_match, venue_id, home_team_id, away_team_id, status)
		VALUES (DATE_ADD(NOW(), INTERVAL %d DAY), DATE_ADD(NOW(), INTERVAL %d DAY), 'okregowa', ?, ?, ?, ?)`, daysFromNow, daysFromNow)
	
	res, err := db.exec(query, venueID, homeTeamID, awayTeamID, status)
	if err != nil {
		t.Fatalf("failed to create test match: %v", err)
	}
	id, _ := res.LastInsertId()

	return int(id), homeTeamID, awayTeamID, func() {
		db.exec(`DELETE FROM matches WHERE id = ?`, id)
		cleanupVenue()
		cleanupAway()
		cleanupHome()
	}
}

func createTestWages(t *testing.T, db *Database) (int, func()) {
	t.Helper()
	roleID, cleanupRole := createTestRoleInMatch(t, db)

	res, err := db.exec(`INSERT INTO wages (match_level, role_in_match, fee, valid_from) VALUES ('okregowa', ?, 150.00, '2020-01-01')`, roleID)
	if err != nil {
		t.Fatalf("failed to create test wages: %v", err)
	}
	id, _ := res.LastInsertId()

	return int(id), func() {
		db.exec(`DELETE FROM wages WHERE id = ?`, id)
		cleanupRole()
	}
}

func createTestMatchAssignment(t *testing.T, db *Database, refereeID, matchID int) (int, func()) {
	t.Helper()
	roleID, cleanupRole := createTestRoleInMatch(t, db)

	res, err := db.exec(`INSERT INTO match_assignments (referee_id, match_id, role, assignment_status) VALUES (?, ?, ?, 'accepted')`, refereeID, matchID, roleID)
	if err != nil {
		t.Fatalf("failed to create test assignment: %v", err)
	}
	id, _ := res.LastInsertId()

	return int(id), func() {
		db.exec(`DELETE FROM match_assignments WHERE id = ?`, id)
		cleanupRole()
	}
}

func createTestPayout(t *testing.T, db *Database, assignmentID int, paidDate time.Time) (int, float64, func()) {
	t.Helper()
	wagesID, cleanupWages := createTestWages(t, db)
	amount := 150.00

	res, err := db.exec(`INSERT INTO payouts (assignment_id, wages_id, amount, status, paid_at) VALUES (?, ?, ?, 'paid', ?)`, assignmentID, wagesID, amount, paidDate)
	if err != nil {
		t.Fatalf("failed to create test payout: %v", err)
	}
	id, _ := res.LastInsertId()

	return int(id), amount, func() {
		db.exec(`DELETE FROM payouts WHERE id = ?`, id)
		cleanupWages()
	}
}
