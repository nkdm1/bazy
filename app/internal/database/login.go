package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/nkdm1/bazy/internal/misc"
	"github.com/nkdm1/bazy/internal/types"
)

// IsUserRegistered() calls GetPasswordHash() and returns true
// if user's password is NOT NULL (registered), otherwise returns false
func (db *Database) IsUserRegistered(email string) (bool, types.ErrorApi) {
	_, _, err := db.GetPasswordHash(email)
	if err != nil {
		if errors.Is(err, types.ErrInvalidEmailOrPassword) ||
			errors.Is(err, types.ErrNullPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetPasswordHash queries the 'users' table by `email` in search of 'password_hash'
func (db *Database) GetPasswordHash(email string) (int, string, types.ErrorApi) {
	row, cancel := db.queryRow(`
	SELECT id, password_hash 
		FROM users 
		WHERE 
			email = ?	AND deleted_at IS NULL;
	`, email)
	defer cancel()

	var id int
	var maybeHash sql.NullString
	if err := row.Scan(&id, &maybeHash); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return -1, "", types.ErrInvalidEmailOrPassword
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout during login: %v", err)
			return -1, "", types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure during login: %v", err)
			return -1, "", types.ErrInternalServer
		}
	}
	if !maybeHash.Valid {
		return -1, "", types.ErrNullPassword
	}
	hash := maybeHash.String
	return id, hash, nil
}

// CreatePendingUser creates a user with no password hash.
// The missing password hash marks them as "pending activation".
// If user already exist, it will return existing user ID with NIL error.
func (db *Database) CreatePendingUser(email, name, surname string) (int, types.ErrorApi) {
	res, err := db.exec(`
		INSERT INTO users (email, name, surname)
		VALUES (?, ?, ?);
	`, email, name, surname)
	if err != nil {
		if e, found := errors.AsType[*mysql.MySQLError](err); found {
			if e.Number == ErrCodeDuplicateEntry {
				return db.GetUserByEmail(email)
			}
		}
		log.Printf("[ERROR]: Database failure during pending user creation: %v", err)
		return -1, types.ErrInternalServer
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("[ERROR]: Database failure during pending user creation: %v", err)
		return -1, types.ErrInternalServer
	}
	return int(id), nil
}

// CreateNewPassword() securely hashes the token before saving it to the set_password table.
// Returns the token in a hex format.
func (db *Database) CreateNewPassword(userId int) (string, types.ErrorApi) {
	token, apiErr := misc.GenerateToken()
	if apiErr != nil {
		return "", apiErr
	}
	tokenHashBytes := sha256.Sum256(token)
	tokenHash := hex.EncodeToString(tokenHashBytes[:])
	_, err := db.exec(`
		INSERT INTO set_password (user_id, token_hash)
		VALUES (?, ?);
		`, userId, tokenHash)
	if err != nil {
		log.Printf("[ERROR]: Database failure during CreateNewPassword: %v", err)
		return "", types.ErrInternalServer
	}

	return hex.EncodeToString(token), nil
}

func (db *Database) GetUserByEmail(email string) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT id 
		FROM users
		WHERE email = ?
	`, email)
	defer cancel()

	var id int
	if err := row.Scan(&id); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return -1, types.ErrNotFound
		case errors.Is(err, context.DeadlineExceeded):
			log.Printf("[ERROR]: Database timeout during GetUserByEmail: %v", err)
			return -1, types.ErrTimeout
		default:
			log.Printf("[ERROR]: Database failure during GetUserByEmail: %v", err)
			return -1, types.ErrInternalServer
		}
	}
	return id, nil

}

// ConsumeRegistrationToken looks up the token hash. If valid, it deletes it
// to prevent reuse (preventing replay attacks) and returns the associated user ID.
func (db *Database) ConsumeRegistrationToken(tokenHash string) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT user_id
		FROM set_password 
		WHERE token_hash = ?
		AND expire_time > now()
		AND status = 'pending';
	`, tokenHash)
	defer cancel()

	var userId int
	if err := row.Scan(&userId); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return -1, types.ErrInvalidToken
		case errors.Is(err, context.DeadlineExceeded):
			return -1, types.ErrTimeout
		default:
			log.Printf("[ERROR] Database failure looking up token: %v", err)
			return -1, types.ErrInternalServer
		}
	}

	_, err := db.exec(`
		UPDATE set_password 
		SET status = 'used'
		WHERE token_hash = ?;
	`, tokenHash)

	if err != nil {
		log.Printf("[ERROR] Database failure updating consumed token: %v", err)
		return -1, types.ErrInternalServer
	}

	return userId, nil
}

// ActivateUserPassword updates the pending user's record with their bcrypt hash,
// formally completing their registration.
func (db *Database) ActivateUserPassword(userId int, bcryptHash string) types.ErrorApi {
	_, err := db.exec(`
		UPDATE users 
		SET password_hash = ? 
		WHERE id = ?;
	`, bcryptHash, userId)

	if err != nil {
		log.Printf("[ERROR] Database failure setting user password: %v", err)
		return types.ErrInternalServer
	}

	return nil
}

// ValidateSession() looks up the database for matching tokenHash
// and returns the associated userId.
// If tokenHash does not match or is expired, returns types.ErrUnauthorized error.
func (db *Database) ValidateSession(tokenHash string) (int, types.ErrorApi) {
	row, cancel := db.queryRow(`
		SELECT user_id 
		FROM auth_tokens 
		WHERE token_hash = ? 
		AND expire_time > now();
	`, tokenHash)
	defer cancel()

	var userId int
	if err := row.Scan(&userId); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return -1, types.ErrUnauthorized
		case errors.Is(err, context.DeadlineExceeded):
			return -1, types.ErrTimeout
		default:
			log.Printf("[ERROR] Database failure validating session: %v", err)
			return -1, types.ErrInternalServer
		}
	}

	_, err := db.exec(`UPDATE auth_tokens SET last_used_at = now() WHERE token_hash = ?`, tokenHash)
	if err != nil {
		log.Printf("[ERROR] Database failure updating token last used timestamp: %v", err)
		return -1, types.ErrInternalServer
	}

	return userId, nil
}

// CreateAuthToken() writes new row to auth_tokens table
func (db *Database) CreateAuthToken(userId int, tokenHash string) types.ErrorApi {
	_, err := db.exec(`
		INSERT INTO auth_tokens (user_id, token_hash)
		VALUES (?, ?)
	`, userId, tokenHash)

	if err != nil {
		log.Printf("[ERROR] Database failure inserting auth token: %v", err)
		return types.ErrInternalServer
	}

	return nil
}
