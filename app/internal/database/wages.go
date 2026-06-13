package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// GetRoleInMatchID looks up a role_in_match row by its enum string value
// and returns the corresponding id.
func (db *Database) GetRoleInMatchID(matchRole string) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT id
		FROM role_in_match
		WHERE match_role = ?;
	`, matchRole)
	defer cancel()

	var id int
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, types.ErrNotFound
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[ERROR]: Database timeout while fetching role in match %q: %v", matchRole, err)
			return -1, types.ErrTimeout
		}
		log.Printf("[ERROR]: Database failure while fetching role in match %q: %v", matchRole, err)
		return -1, types.ErrInternalServer
	}
	return id, nil
}

// InsertWage inserts a new row into the wages table with valid_from set to
// the current date (CURDATE()), preserving the historical fee records.
func (db *Database) InsertWage(matchLevel string, roleInMatchID int, fee float64) types.ErrorApi {
	_, err := db.exec(`
		INSERT INTO wages (match_level, role_in_match, fee, valid_from)
		VALUES (?, ?, ?, CURDATE());
	`, matchLevel, roleInMatchID, fee)
	if err != nil {
		log.Printf("[ERROR]: Database failure inserting wage: %v", err)
		return types.ErrInternalServer
	}
	return nil
}
