package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/nkdm1/bazy/internal/misc"
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

type RefereeDirectoryEntry struct {
	FirstName    string `json:"first_name"`
	Surname      string `json:"surname"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Postcode     string `json:"postcode"`
	City         string `json:"city"`
	Street       string `json:"street"`
	StreetNumber string `json:"street_number"`
	FlatNumber   string `json:"flat_number"`
}

// GetRefereeDirectory retrieves a list of all referees along with their user
// information and address details.
func (db *Database) GetRefereeDirectory() ([]RefereeDirectoryEntry, types.ErrorApi) {
	rows, cancel, err := db.query(`
		SELECT
			u.name, u.surname, u.email, COALESCE(r.phone, ''),
			a.postcode, a.city, COALESCE(a.street, ''), a.street_number, COALESCE(a.flat_number, '')
		FROM referees r
		JOIN users u ON r.user_id = u.id
		JOIN address a ON r.address_id = a.id;
	`)
	defer cancel()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[ERROR]: Database timeout while fetching referee directory: %v", err)
			return nil, types.ErrTimeout
		}
		log.Printf("[ERROR]: Database failure while fetching referee directory: %v", err)
		return nil, types.ErrInternalServer
	}
	defer rows.Close()

	var list []RefereeDirectoryEntry
	for rows.Next() {
		var entry RefereeDirectoryEntry
		if err := rows.Scan(
			&entry.FirstName,
			&entry.Surname,
			&entry.Email,
			&entry.Phone,
			&entry.Postcode,
			&entry.City,
			&entry.Street,
			&entry.StreetNumber,
			&entry.FlatNumber,
		); err != nil {
			log.Printf("[ERROR]: Failed to scan referee directory entry: %v", err)
			return nil, types.ErrInternalServer
		}
		list = append(list, entry)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR]: Row iteration error while fetching referee directory: %v", err)
		return nil, types.ErrInternalServer
	}

	return list, nil
}

type LicenseEntry struct {
	LicenseNumber string    `json:"license_number"`
	LicenseName   string    `json:"license_name"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpireAt      time.Time `json:"expire_at"`
}

type RefereeProfile struct {
	Name         string         `json:"name"`
	Surname      string         `json:"surname"`
	Email        string         `json:"email"`
	Phone        string         `json:"phone"`
	Postcode     string         `json:"postcode"`
	City         string         `json:"city"`
	Street       string         `json:"street"`
	StreetNumber string         `json:"street_number"`
	FlatNumber   string         `json:"flat_number"`
	Licenses     []LicenseEntry `json:"licenses"`
}

// GetRefereeProfile fetches full details for the given user, including their
// personal data, referee address, and active licenses.
func (db *Database) GetRefereeProfile(userID int) (*RefereeProfile, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT
			u.name, u.surname, u.email, COALESCE(r.phone, ''),
			a.postcode, a.city, COALESCE(a.street, ''), a.street_number, COALESCE(a.flat_number, ''),
			r.id
		FROM users u
		JOIN referees r ON u.id = r.user_id
		JOIN address a ON r.address_id = a.id
		WHERE u.id = ? AND u.deleted_at IS NULL;
	`, userID)
	defer cancel()

	var profile RefereeProfile
	var refereeID int
	if err := row.Scan(
		&profile.Name,
		&profile.Surname,
		&profile.Email,
		&profile.Phone,
		&profile.Postcode,
		&profile.City,
		&profile.Street,
		&profile.StreetNumber,
		&profile.FlatNumber,
		&refereeID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrNotFound
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[ERROR]: Database timeout while fetching referee profile %d: %v", userID, err)
			return nil, types.ErrTimeout
		}
		log.Printf("[ERROR]: Database failure while fetching referee profile %d: %v", userID, err)
		return nil, types.ErrInternalServer
	}

	// Fetch licenses
	rows, cancelL, err := db.query(`
		SELECT l.license_number, ln.license_name, l.issued_at, l.expire_at
		FROM licenses l
		JOIN licenses_names ln ON l.license_name_id = ln.id
		WHERE l.referee_id = ?;
	`, refereeID)
	defer cancelL()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[ERROR]: Database timeout while fetching licenses for referee %d: %v", refereeID, err)
			return nil, types.ErrTimeout
		}
		log.Printf("[ERROR]: Database failure while fetching licenses for referee %d: %v", refereeID, err)
		return nil, types.ErrInternalServer
	}
	defer rows.Close()

	profile.Licenses = []LicenseEntry{}
	for rows.Next() {
		var lic LicenseEntry
		if err := rows.Scan(&lic.LicenseNumber, &lic.LicenseName, &lic.IssuedAt, &lic.ExpireAt); err != nil {
			log.Printf("[ERROR]: Failed to scan license entry for referee %d: %v", refereeID, err)
			return nil, types.ErrInternalServer
		}
		profile.Licenses = append(profile.Licenses, lic)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR]: Row iteration error while fetching licenses for referee %d: %v", refereeID, err)
		return nil, types.ErrInternalServer
	}

	return &profile, nil
}

// GetLicenseNameID looks up the ID of a license name, or returns ErrNotFound if it doesn't exist.
func (db *Database) GetLicenseNameID(licenseName string) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT id FROM licenses_names WHERE license_name = ?;
	`, licenseName)
	defer cancel()

	var id int
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, types.ErrNotFound
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[ERROR]: Database timeout fetching license name %q: %v", licenseName, err)
			return -1, types.ErrTimeout
		}
		log.Printf("[ERROR]: Database failure fetching license name %q: %v", licenseName, err)
		return -1, types.ErrInternalServer
	}
	return id, nil
}

// InsertLicense adds a verified license record into the licenses table.
func (db *Database) InsertLicense(refereeID, licenseNameID int, licenseNumber string, issuedAt, expireAt time.Time) types.ErrorApi {
	_, err := db.exec(`
		INSERT INTO licenses (referee_id, license_number, license_name_id, issued_at, expire_at)
		VALUES (?, ?, ?, ?, ?);
	`, refereeID, licenseNumber, licenseNameID, issuedAt, expireAt)
	if err != nil {
		log.Printf("[ERROR]: Database failure inserting license: %v", err)
		return types.ErrInternalServer
	}
	return nil
}

// CreatePhoneChangeToken generates a verification token and records the phone change request.
func (db *Database) CreatePhoneChangeToken(refereeID int, newPhone string) (string, types.ErrorApi) {
	tokenBytes, apiErr := misc.GenerateToken()
	if apiErr != nil {
		return "", apiErr
	}
	tokenHashBytes := sha256.Sum256(tokenBytes)
	tokenHash := hex.EncodeToString(tokenHashBytes[:])

	_, err := db.exec(`
		INSERT INTO set_phone (referee_id, new_phone, token_hash)
		VALUES (?, ?, ?);
	`, refereeID, newPhone, tokenHash)
	if err != nil {
		log.Printf("[ERROR]: Database failure inserting set_phone: %v", err)
		return "", types.ErrInternalServer
	}

	return hex.EncodeToString(tokenBytes), nil
}

// ConsumePhoneChangeToken processes the verification token and applies the new phone.
func (db *Database) ConsumePhoneChangeToken(tokenHash string) types.ErrorApi {
	row, cancel := db.queryRow(`
		SELECT id, referee_id, new_phone, expire_time, status
		FROM set_phone
		WHERE token_hash = ?;
	`, tokenHash)
	defer cancel()

	var id, refereeID int
	var newPhone string
	var expireTime time.Time
	var status string

	if err := row.Scan(&id, &refereeID, &newPhone, &expireTime, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrInvalidToken
		}
		log.Printf("[ERROR]: Database failure finding set_phone token: %v", err)
		return types.ErrInternalServer
	}

	if status == "used" || status == "expired" {
		return types.ErrInvalidToken
	}

	if time.Now().After(expireTime) {
		_, err := db.exec(`UPDATE set_phone SET status = 'expired' WHERE id = ?`, id)
		if err != nil {
			log.Printf("[ERROR]: Database failure marking set_phone token expired: %v", err)
		}
		return types.ErrInvalidToken
	}

	// Verify referee exists in referees table
	var refereeExists int
	rowRef, cancelRef := db.queryRow(`SELECT COUNT(*) FROM referees WHERE id = ?`, refereeID)
	if err := rowRef.Scan(&refereeExists); err != nil {
		cancelRef()
		log.Printf("[ERROR]: Database failure checking referee existence: %v", err)
		return types.ErrInternalServer
	}
	cancelRef()

	if refereeExists > 0 {
		_, err := db.exec(`UPDATE referees SET phone = ? WHERE id = ?`, newPhone, refereeID)
		if err != nil {
			log.Printf("[ERROR]: Database failure updating phone in referees: %v", err)
			return types.ErrInternalServer
		}
	}

	_, err := db.exec(`UPDATE set_phone SET status = 'used' WHERE id = ?`, id)
	if err != nil {
		log.Printf("[ERROR]: Database failure marking set_phone token used: %v", err)
		return types.ErrInternalServer
	}

	return nil
}


