package queries

import (
	"database/sql"
)

func getPasswordHash(db *sql.DB) (string, error) {
	row := db.QueryRow(`
	SELECT id, password_hash 
		FROM users 
		WHERE email = ? 
			AND deleted_at IS NULL;
	`, )
	var hash string
	if err := row.Scan(hash); err != nil {
		return "", err
	} 

	return hash, nil
}
