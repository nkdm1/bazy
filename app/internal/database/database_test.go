package database

import (
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// --- CENTRALIZED TEST FIXTURES ---
// Accessible by any *_test.go file in this directory!
var (
	testAddress = struct {
		ID           int
		Postcode     string
		City         string
		Street       string
		StreetNumber string
	}{
		ID:           9999,
		Postcode:     "00-001",
		City:         "Warszawa",
		Street:       "Marszałkowska",
		StreetNumber: "1",
	}

	testUser = struct {
		ID           int
		Email        string
		PasswordHash string
		Name         string
		Surname      string
		Role         string
	}{
		ID:           9999,
		Email:        "test@test.com",
		PasswordHash: "hash",
		Name:         "Jan",
		Surname:      "Kowalski",
		Role:         "referee",
	}

	testReferee = struct {
		ID        int
		UserID    int
		AddressID int
	}{
		ID:        9999,
		UserID:    9999, // testUser.ID
		AddressID: 9999, // testAddress.ID
	}

	testAvailability = struct {
		ID            int
		RefereeID     int
		AvailableDate time.Time
	}{
		ID:            9999,
		RefereeID:     9999, // testReferee.ID
		AvailableDate: time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
	}

	testTeamA = struct {
		ID   int
		Name string
		City string
	}{
		ID:   9998,
		Name: "Team A",
		City: "Warszawa",
	}

	testTeamB = struct {
		ID   int
		Name string
		City string
	}{
		ID:   9999,
		Name: "Team B",
		City: "Kraków",
	}

	testVenue = struct {
		ID        int
		GymName   string
		AddressID int
	}{
		ID:        9999,
		GymName:   "Hala Główna",
		AddressID: 9999, // testAddress.ID
	}

	testMatchLevel = struct {
		ID         int
		MatchLevel string
	}{
		ID:         9999,
		MatchLevel: "okregowa",
	}

	testRoleInMatch = struct {
		ID        int
		MatchRole string
	}{
		ID:        9999,
		MatchRole: "umpire",
	}

	testMatchUpcoming = struct {
		ID           int
		LevelOfMatch int
		VenueID      int
		HomeTeamID   int
		AwayTeamID   int
		Status       string
	}{
		ID:           9998,
		LevelOfMatch: 9999, // testMatchLevel.ID
		VenueID:      9999, // testVenue.ID
		HomeTeamID:   9998, // testTeamA.ID
		AwayTeamID:   9999, // testTeamB.ID
		Status:       "scheduled",
	}

	testMatchFar = struct {
		ID           int
		LevelOfMatch int
		VenueID      int
		HomeTeamID   int
		AwayTeamID   int
		Status       string
	}{
		ID:           9999,
		LevelOfMatch: 9999, // testMatchLevel.ID
		VenueID:      9999, // testVenue.ID
		HomeTeamID:   9998, // testTeamA.ID
		AwayTeamID:   9999, // testTeamB.ID
		Status:       "scheduled",
	}

	testWages = struct {
		ID          int
		MatchLevel  int
		RoleInMatch int
		Fee         float64
		ValidFrom   time.Time
	}{
		ID:          9999,
		MatchLevel:  9999, // testMatchLevel.ID
		RoleInMatch: 9999, // testRoleInMatch.ID
		Fee:         150.00,
		ValidFrom:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	testMatchAssignment = struct {
		ID               int
		RefereeID        int
		MatchID          int
		Role             int
		AssignmentStatus string
	}{
		ID:               9999,
		RefereeID:        9999, // testReferee.ID
		MatchID:          9998, // testMatchUpcoming.ID
		Role:             9999, // testRoleInMatch.ID
		AssignmentStatus: "accepted",
	}

	testPayout = struct {
		ID           int
		AssignmentID int
		WagesID      int
		Amount       float64
		Status       string
		PaidAt       time.Time
	}{
		ID:           9999,
		AssignmentID: 9999, // testMatchAssignment.ID
		WagesID:      9999, // testWages.ID
		Amount:       150.00,
		Status:       "paid",
		PaidAt:       time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC),
	}
)

func testDB(t *testing.T) *Database {
	t.Helper()

	db := Init()
	seedTestData(t, db)
	t.Cleanup(func() {
		cleanTestData(t, db)
		db.instance.Close()
	})

	return db
}

func seedTestData(t *testing.T, db *Database) {
	t.Helper()

	// 1. Address
	_, err := db.exec(`INSERT INTO address (id, postcode, city, street, street_number) VALUES (?, ?, ?, ?, ?)`,
		testAddress.ID, testAddress.Postcode, testAddress.City, testAddress.Street, testAddress.StreetNumber)
	if err != nil {
		t.Fatalf("seed address failed: %v", err)
	}

	// 2. Users
	_, err = db.exec(`INSERT INTO users (id, email, password_hash, name, surname, role) VALUES (?, ?, ?, ?, ?, ?)`,
		testUser.ID, testUser.Email, testUser.PasswordHash, testUser.Name, testUser.Surname, testUser.Role)
	if err != nil {
		t.Fatalf("seed users failed: %v", err)
	}

	// 3. Referees
	_, err = db.exec(`INSERT INTO referees (id, user_id, address_id) VALUES (?, ?, ?)`,
		testReferee.ID, testReferee.UserID, testReferee.AddressID)
	if err != nil {
		t.Fatalf("seed referees failed: %v", err)
	}

	// 4. Availability
	_, err = db.exec(`INSERT INTO availability (id, referee_id, available_date) VALUES (?, ?, ?)`,
		testAvailability.ID, testAvailability.RefereeID, testAvailability.AvailableDate)
	if err != nil {
		t.Fatalf("seed availability failed: %v", err)
	}

	// 5. Teams (A and B)
	_, err = db.exec(`INSERT INTO teams (id, name, city) VALUES (?, ?, ?)`,
		testTeamA.ID, testTeamA.Name, testTeamA.City)
	if err != nil {
		t.Fatalf("seed team A failed: %v", err)
	}
	_, err = db.exec(`INSERT INTO teams (id, name, city) VALUES (?, ?, ?)`,
		testTeamB.ID, testTeamB.Name, testTeamB.City)
	if err != nil {
		t.Fatalf("seed team B failed: %v", err)
	}

	// 6. Venues
	_, err = db.exec(`INSERT INTO venues (id, gym_name, address_id) VALUES (?, ?, ?)`,
		testVenue.ID, testVenue.GymName, testVenue.AddressID)
	if err != nil {
		t.Fatalf("seed venues failed: %v", err)
	}

	// 7. Matches Level
	_, err = db.exec(`INSERT INTO matches_level (id, match_level) VALUES (?, ?)`,
		testMatchLevel.ID, testMatchLevel.MatchLevel)
	if err != nil {
		t.Fatalf("seed matches_level failed: %v", err)
	}

	// 8. Role In Match
	_, err = db.exec(`INSERT INTO role_in_match (id, match_role) VALUES (?, ?)`,
		testRoleInMatch.ID, testRoleInMatch.MatchRole)
	if err != nil {
		t.Fatalf("seed role_in_match failed: %v", err)
	}

	// 9. Matches (Preserves DATE_ADD evaluation directly inside MySQL while injecting structural IDs)
	_, err = db.exec(`INSERT INTO matches (id, match_start, match_end, level_of_match, venue_id, home_team_id, away_team_id, status)
		 VALUES (?, DATE_ADD(NOW(), INTERVAL 2 DAY), DATE_ADD(NOW(), INTERVAL 2 DAY), ?, ?, ?, ?, ?)`,
		testMatchUpcoming.ID, testMatchUpcoming.LevelOfMatch, testMatchUpcoming.VenueID, testMatchUpcoming.HomeTeamID, testMatchUpcoming.AwayTeamID, testMatchUpcoming.Status)
	if err != nil {
		t.Fatalf("seed upcoming match failed: %v", err)
	}

	_, err = db.exec(`INSERT INTO matches (id, match_start, match_end, level_of_match, venue_id, home_team_id, away_team_id, status)
		 VALUES (?, DATE_ADD(NOW(), INTERVAL 10 DAY), DATE_ADD(NOW(), INTERVAL 10 DAY), ?, ?, ?, ?, ?)`,
		testMatchFar.ID, testMatchFar.LevelOfMatch, testMatchFar.VenueID, testMatchFar.HomeTeamID, testMatchFar.AwayTeamID, testMatchFar.Status)
	if err != nil {
		t.Fatalf("seed far match failed: %v", err)
	}

	// 10. Wages
	_, err = db.exec(`INSERT INTO wages (id, match_level, role_in_match, fee, valid_from) VALUES (?, ?, ?, ?, ?)`,
		testWages.ID, testWages.MatchLevel, testWages.RoleInMatch, testWages.Fee, testWages.ValidFrom)
	if err != nil {
		t.Fatalf("seed wages failed: %v", err)
	}

	// 11. Match Assignments
	_, err = db.exec(`INSERT INTO match_assignments (id, referee_id, match_id, role, assignment_status) VALUES (?, ?, ?, ?, ?)`,
		testMatchAssignment.ID, testMatchAssignment.RefereeID, testMatchAssignment.MatchID, testMatchAssignment.Role, testMatchAssignment.AssignmentStatus)
	if err != nil {
		t.Fatalf("seed match_assignments failed: %v", err)
	}

	// 12. Payouts
	_, err = db.exec(`INSERT INTO payouts (id, assignment_id, wages_id, amount, status, paid_at) VALUES (?, ?, ?, ?, ?, ?)`,
		testPayout.ID, testPayout.AssignmentID, testPayout.WagesID, testPayout.Amount, testPayout.Status, testPayout.PaidAt)
	if err != nil {
		t.Fatalf("seed payouts failed: %v", err)
	}
}

func cleanTestData(t *testing.T, db *Database) {
	t.Helper()

	// Direct child-to-parent execution array to strictly respect Foreign Keys
	cleanups := []struct {
		query string
		arg   any
	}{
		{query: "DELETE FROM payouts WHERE id = ?", arg: testPayout.ID},
		{query: "DELETE FROM match_assignments WHERE id = ?", arg: testMatchAssignment.ID},
		{query: "DELETE FROM wages WHERE id = ?", arg: testWages.ID},
		{query: "DELETE FROM matches WHERE id = ?", arg: testMatchUpcoming.ID},
		{query: "DELETE FROM matches WHERE id = ?", arg: testMatchFar.ID},
		{query: "DELETE FROM role_in_match WHERE id = ?", arg: testRoleInMatch.ID},
		{query: "DELETE FROM matches_level WHERE id = ?", arg: testMatchLevel.ID},
		{query: "DELETE FROM venues WHERE id = ?", arg: testVenue.ID},
		{query: "DELETE FROM teams WHERE id = ?", arg: testTeamA.ID},
		{query: "DELETE FROM teams WHERE id = ?", arg: testTeamB.ID},
		{query: "DELETE FROM availability WHERE id = ?", arg: testAvailability.ID},
		{query: "DELETE FROM referees WHERE id = ?", arg: testReferee.ID},
		{query: "DELETE FROM users WHERE id = ?", arg: testUser.ID},
		{query: "DELETE FROM address WHERE id = ?", arg: testAddress.ID},
	}

	for _, c := range cleanups {
		if _, err := db.exec(c.query, c.arg); err != nil {
			t.Logf("cleanup warning for query [%s]: %v", c.query, err)
		}
	}
}
