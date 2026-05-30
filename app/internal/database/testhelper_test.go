package database

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// testDB tworzy połączenie z bazą testową.
// DSN czytamy ze zmiennej środowiskowej TEST_DSN,
// np. "root:root@tcp(localhost:3306)/bazy_test"
func testDB(t *testing.T) *Database {
	t.Helper()

	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set, skipping integration test")
	}

	sqlDB, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	db := &Database{instance: sqlDB}

	// Czyścimy i seedujemy bazę przed każdym testem
	seedTestData(t, sqlDB)

	// Po teście czyścimy dane
	t.Cleanup(func() {
		cleanTestData(t, sqlDB)
		sqlDB.Close()
	})

	return db
}

func seedTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	queries := []string{
		// Adresy
		`INSERT INTO address (id, postcode, city, street, street_number)
		 VALUES (1, '00-001', 'Warszawa', 'Marszałkowska', '1')`,

		// Użytkownicy
		`INSERT INTO users (id, email, password_hash, name, surname, role)
		 VALUES (1, 'test@test.com', 'hash', 'Jan', 'Kowalski', 'referee')`,

		// Sędzia
		`INSERT INTO referees (id, user_id, address_id)
		 VALUES (1, 1, 1)`,

		// Dostępność
		`INSERT INTO availability (id, referee_id, available_date)
		 VALUES (1, 1, '2026-06-20')`,

		// Drużyny
		`INSERT INTO teams (id, name, city) VALUES (1, 'Team A', 'Warszawa')`,
		`INSERT INTO teams (id, name, city) VALUES (2, 'Team B', 'Kraków')`,

		// Venue
		`INSERT INTO venues (id, gym_name, address_id) VALUES (1, 'Hala Główna', 1)`,

		// Poziom meczu
		`INSERT INTO matches_level (id, match_level) VALUES (1, 'okregowa')`,

		// Rola w meczu
		`INSERT INTO role_in_match (id, match_role) VALUES (1, 'umpire')`,

		// Mecze - jeden w nadchodzącym tygodniu, jeden poza
		`INSERT INTO matches (id, match_start, match_end, level_of_match, venue_id, home_team_id, away_team_id, status)
		 VALUES (1, DATE_ADD(NOW(), INTERVAL 2 DAY), DATE_ADD(NOW(), INTERVAL 2 DAY), 1, 1, 1, 2, 'scheduled')`,

		`INSERT INTO matches (id, match_start, match_end, level_of_match, venue_id, home_team_id, away_team_id, status)
		 VALUES (2, DATE_ADD(NOW(), INTERVAL 10 DAY), DATE_ADD(NOW(), INTERVAL 10 DAY), 1, 1, 1, 2, 'scheduled')`,

		// Stawki
		`INSERT INTO wages (id, match_level, role_in_match, fee, valid_from)
		 VALUES (1, 1, 1, 150.00, '2026-01-01')`,

		// Przypisanie sędziego do meczu 1
		`INSERT INTO match_assignments (id, referee_id, match_id, role, assignment_status)
		 VALUES (1, 1, 1, 1, 'accepted')`,

		// Wypłata - zapłacona
		`INSERT INTO payouts (id, assignment_id, wages_id, amount, status, paid_at)
		 VALUES (1, 1, 1, 150.00, 'paid', '2026-06-15 12:00:00')`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed failed: %v\nquery: %s", err, q)
		}
	}
}

func cleanTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Kolejność ważna — najpierw usuwamy tabele zależne
	tables := []string{
		"payouts", "match_assignments", "reviews",
		"matches", "wages", "availability",
		"licenses", "referees", "set_phone",
		"venues", "teams", "matches_level",
		"role_in_match", "auth_tokens", "set_mail",
		"set_password", "users", "address",
	}

	for _, table := range tables {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Logf("cleanup warning for table %s: %v", table, err)
		}
	}
}
