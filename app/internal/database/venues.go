package database

import (
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// CreateVenue inserts address details first, retrieves the address_id, and then inserts the venue.
func (db *Database) CreateVenue(gymName, postcode, city, street, streetNumber, flatNumber string) types.ErrorApi {
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
		log.Printf("[ERROR]: Failed to insert address for venue %q: %v", gymName, err)
		return types.ErrInternalServer
	}

	addressID, err := result.LastInsertId()
	if err != nil {
		log.Printf("[ERROR]: Failed to retrieve address ID for venue %q: %v", gymName, err)
		return types.ErrInternalServer
	}

	_, err = db.exec(`
		INSERT INTO venues (gym_name, address_id)
		VALUES (?, ?);
	`, gymName, addressID)
	if err != nil {
		log.Printf("[ERROR]: Failed to insert venue %q: %v", gymName, err)
		return types.ErrInternalServer
	}

	return nil
}
