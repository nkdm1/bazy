package database

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/nkdm1/bazy/internal/types"
)

// GetUserRole fetches the role of a user from the users table.
func (db *Database) GetUserRole(userID int) (string, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT role
		FROM users
		WHERE id = ?;
	`, userID)
	defer cancel()

	var role string
	if err := row.Scan(&role); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", types.ErrNotFound
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout while fetching user role %d: %v", userID, err)
			return "", types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure while fetching user role %d: %v", userID, err)
			return "", types.ErrInternalServer
		}
	}
	return role, nil
}

// SoftDeleteUser sets deleted_at to NOW() for the given user, effectively
// disabling their account while preserving all historical records.
func (db *Database) SoftDeleteUser(userID int) types.ErrorApi {
	result, err := db.exec(`
		UPDATE users
		SET deleted_at = NOW()
		WHERE id = ? AND deleted_at IS NULL;
	`, userID)
	if err != nil {
		log.Printf("[ERROR]: Database failure soft-deleting user %d: %v", userID, err)
		return types.ErrInternalServer
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[ERROR]: Could not retrieve rows affected for SoftDeleteUser %d: %v", userID, err)
		return types.ErrInternalServer
	}
	if rowsAffected == 0 {
		return types.ErrNotFound
	}

	return nil
}

// InvalidateAllUserSessions deletes all auth_tokens for the given user,
// immediately logging them out of every active session.
func (db *Database) InvalidateAllUserSessions(userID int) types.ErrorApi {
	_, err := db.exec(`
		DELETE FROM auth_tokens
		WHERE user_id = ?;
	`, userID)
	if err != nil {
		log.Printf("[ERROR]: Database failure invalidating sessions for user %d: %v", userID, err)
		return types.ErrInternalServer
	}
	return nil
}

// UpdateUserProfile updates a user's phone and address.
func (db *Database) UpdateUserProfile(userID int, phone, postcode, city, street, streetNumber, flatNumber string) types.ErrorApi {
	var flatNumPtr *string
	if flatNumber != "" {
		flatNumPtr = &flatNumber
	}
	var streetPtr *string
	if street != "" {
		streetPtr = &street
	}
	var phonePtr *string
	if phone != "" {
		phonePtr = &phone
	}
	var postcodePtr *string
	if postcode != "" {
		postcodePtr = &postcode
	}
	var cityPtr *string
	if city != "" {
		cityPtr = &city
	}
	var stNumPtr *string
	if streetNumber != "" {
		stNumPtr = &streetNumber
	}

	// check if user has address_id
	var addressID sql.NullInt64
	row, cancel := db.queryRow(`SELECT address_id FROM users WHERE id = ?`, userID)
	err := row.Scan(&addressID)
	cancel()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		return types.ErrInternalServer
	}

	if addressID.Valid {
		// update existing address
		_, err = db.exec(`
			UPDATE address
			SET postcode = ?, city = ?, street = ?, street_number = ?, flat_number = ?
			WHERE id = ?;
		`, postcodePtr, cityPtr, streetPtr, stNumPtr, flatNumPtr, addressID.Int64)
		if err != nil {
			return types.ErrInternalServer
		}
	} else {
		// insert new address
		res, err := db.exec(`
			INSERT INTO address (postcode, city, street, street_number, flat_number)
			VALUES (?, ?, ?, ?, ?);
		`, postcodePtr, cityPtr, streetPtr, stNumPtr, flatNumPtr)
		if err != nil {
			return types.ErrInternalServer
		}
		newAddressID, _ := res.LastInsertId()
		_, err = db.exec(`UPDATE users SET address_id = ? WHERE id = ?`, newAddressID, userID)
		if err != nil {
			return types.ErrInternalServer
		}
	}

	_, err = db.exec(`UPDATE users SET phone = ? WHERE id = ?`, phonePtr, userID)
	if err != nil {
		return types.ErrInternalServer
	}
	return nil
}

// ApplyReferee attempts to add a user to the referees table
func (db *Database) ApplyReferee(userID int) types.ErrorApi {
	row, cancel := db.queryRow("SELECT phone, address_id FROM users WHERE id = ?", userID)
	defer cancel()

	var phone sql.NullString
	var addressID sql.NullInt64
	if err := row.Scan(&phone, &addressID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNotFound
		}
		return types.ErrInternalServer
	}

	if !phone.Valid || phone.String == "" || !addressID.Valid {
		return types.ErrInvalidPayload // incomplete profile
	}
	
	// Also check if the address actually has city, postcode, etc.
	var postcode, city, streetNum sql.NullString
	addrRow, addrCancel := db.queryRow("SELECT postcode, city, street_number FROM address WHERE id = ?", addressID.Int64)
	if err := addrRow.Scan(&postcode, &city, &streetNum); err != nil {
		addrCancel()
		return types.ErrInternalServer
	}
	addrCancel()
	
	if !postcode.Valid || postcode.String == "" || !city.Valid || city.String == "" || !streetNum.Valid || streetNum.String == "" {
		return types.ErrInvalidPayload // incomplete profile
	}

	_, err := db.exec("INSERT IGNORE INTO referees (user_id) VALUES (?)", userID)
	if err != nil {
		return types.ErrInternalServer
	}

	_, err = db.exec("UPDATE users SET role = 'referee' WHERE id = ?", userID)
	if err != nil {
		return types.ErrInternalServer
	}

	return nil
}
