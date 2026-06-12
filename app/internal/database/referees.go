package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

func (db *Database) GetRefereeIDByUserID(userID int) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT id
		FROM referees
		WHERE user_id = ?;
	`, userID)
	defer cancel()

	var refereeID int
	if err := row.Scan(&refereeID); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return -1, types.ErrNotFound
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching referee by user_id %d: %v", userID, err)
			return -1, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching referee by user_id %d: %v", userID, err)
			return -1, types.ErrInternalServer
		}
	}
	return refereeID, nil
}

// SetUserAsReferee inserts address data and promotes a user to a referee
func (db *Database) SetUserAsReferee(userID int, phone, postcode, city, street, streetNumber, flatNumber string) types.ErrorApi {
	// First, insert the address
	var flatNumPtr *string
	if flatNumber != "" {
		flatNumPtr = &flatNumber
	}
	var streetPtr *string
	if street != "" {
		streetPtr = &street
	}

	result, err := db.exec(`
		INSERT INTO address (postcode, city, street, street_number, flat_number)
		VALUES (?, ?, ?, ?, ?);
	`, postcode, city, streetPtr, streetNumber, flatNumPtr)

	if err != nil {
		log.Printf("[ERROR]: Failed to insert address for user %d: %v", userID, err)
		return types.ErrInternalServer
	}

	addressID, err := result.LastInsertId()
	if err != nil {
		log.Printf("[ERROR]: Failed to retrieve address ID for user %d: %v", userID, err)
		return types.ErrInternalServer
	}

	// Insert into referees table
	_, err = db.exec(`
		INSERT INTO referees (user_id, address_id, phone)
		VALUES (?, ?, ?);
	`, userID, addressID, phone)

	if err != nil {
		log.Printf("[ERROR]: Failed to insert referee record for user %d: %v", userID, err)
		return types.ErrInternalServer
	}

	// Optionally update user role to 'referee'
	_, err = db.exec(`
		UPDATE users SET role = 'referee' WHERE id = ?;
	`, userID)
	if err != nil {
		log.Printf("[ERROR]: Failed to update user role to referee for user %d: %v", userID, err)
		return types.ErrInternalServer
	}

	return nil
}
