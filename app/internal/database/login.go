package database

func (db Database) GetPasswordHash(email string) (string, error) {
	row := db.instance.QueryRow(`
	SELECT id, password_hash 
		FROM users 
		WHERE email = ? 
			AND deleted_at IS NULL;
	`, email)
	var hash string
	if err := row.Scan(hash); err != nil {
		return "", err
	} 

	return hash, nil
}
