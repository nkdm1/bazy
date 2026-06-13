package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:root@tcp(ubuntu:3306)/db?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// insert a referee
	res, err := db.Exec("INSERT INTO users (email, password_hash, name, surname, role) VALUES ('ref@example.com', '$2a$10$oFWwM3dfY.FqK2bMzpmVUOeDzTBEIxiPfESmdAyFpoKjwIp8I916i', 'r', 'r', 'referee') ON DUPLICATE KEY UPDATE password_hash='$2a$10$oFWwM3dfY.FqK2bMzpmVUOeDzTBEIxiPfESmdAyFpoKjwIp8I916i'")
	if err != nil {
		log.Printf("users error: %v", err)
	}
	var uID int
	db.QueryRow("SELECT id FROM users WHERE email = 'ref@example.com'").Scan(&uID)

	res, err = db.Exec("INSERT INTO address (postcode, city, street, street_number) VALUES ('00-000', 'c1', 'a1', '1')")
	addr1, _ := res.LastInsertId()

	db.Exec("INSERT IGNORE INTO referees (user_id, address_id) VALUES (?, ?)", uID, addr1)
	var refID int
	db.QueryRow("SELECT id FROM referees WHERE user_id = ?", uID).Scan(&refID)

	// insert role
	db.Exec("INSERT IGNORE INTO role_in_match (match_role) VALUES ('crew_chief')")
	var roleID int
	db.QueryRow("SELECT id FROM role_in_match WHERE match_role = 'crew_chief'").Scan(&roleID)

	// insert wage
	db.Exec("INSERT IGNORE INTO wages (match_level, role_in_match, fee, valid_from) VALUES ('okregowa', ?, 150.00, '2020-01-01')", roleID)

	// insert teams & venue
	res, err = db.Exec("INSERT INTO teams (name, city) VALUES ('t1', 'c1')")
	if err != nil { log.Fatal(err) }
	t1, _ := res.LastInsertId()
	res, err = db.Exec("INSERT INTO teams (name, city) VALUES ('t2', 'c2')")
	if err != nil { log.Fatal(err) }
	t2, _ := res.LastInsertId()
	res, err = db.Exec("INSERT INTO address (postcode, city, street, street_number) VALUES ('00-000', 'c1', 'a1', '1')")
	if err != nil { log.Printf("address error: %v", err) }
	a1, _ := res.LastInsertId()
	res, err = db.Exec("INSERT INTO venues (gym_name, address_id) VALUES ('v1', ?)", a1)
	if err != nil { log.Printf("venues error: %v", err) }
	v1, _ := res.LastInsertId()

	// insert match
	res, _ = db.Exec("INSERT INTO matches (match_start, match_end, level_of_match, venue_id, home_team_id, away_team_id, status) VALUES (DATE_ADD(NOW(), INTERVAL -1 DAY), DATE_ADD(NOW(), INTERVAL -1 DAY), 'okregowa', ?, ?, ?, 'scheduled')", v1, t1, t2)
	mID, _ := res.LastInsertId()

	// insert assignment
	db.Exec("INSERT INTO match_assignments (referee_id, match_id, role, assignment_status) VALUES (?, ?, ?, 'accepted')", refID, mID, roleID)

	fmt.Printf("{\"match_id\": %d, \"referee_id\": %d}\n", mID, refID)
}
