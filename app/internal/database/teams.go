package database

import (
	"errors"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/nkdm1/bazy/internal/types"
)

// CreateTeam inserts a new team into the teams table.
// Returns ErrConflict if a team with the same name and city already exists.
func (db *Database) CreateTeam(name, city string) types.ErrorApi {
	row, cancel := db.queryRow(`
		SELECT COUNT(*)
		FROM teams
		WHERE name = ? AND city = ?;
	`, name, city)
	defer cancel()

	var count int
	if err := row.Scan(&count); err != nil {
		log.Printf("[ERROR]: Database failure checking team uniqueness: %v", err)
		return types.ErrInternalServer
	}
	if count > 0 {
		return types.ErrConflict
	}

	_, err := db.exec(`
		INSERT INTO teams (name, city)
		VALUES (?, ?);
	`, name, city)
	if err != nil {
		if mysqlErr, ok := errors.AsType[*mysql.MySQLError](err); ok {
			if mysqlErr.Number == ErrCodeDuplicateEntry {
				return types.ErrConflict
			}
		}
		log.Printf("[ERROR]: Database failure creating team %q: %v", name, err)
		return types.ErrInternalServer
	}
	return nil
}


