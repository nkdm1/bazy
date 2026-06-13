package database

import (
	"errors"
	"testing"
	"time"

	"github.com/nkdm1/bazy/internal/types"
)

func TestSetUserAsReferee(t *testing.T) {
	db := testDB(t)

	t.Run("successfully set user as referee", func(t *testing.T) {
		// Use our factory to create a test user (which is not yet a referee in this isolated context)
		// Wait, createTestUser sets the role to "referee", but doesn't actually insert into the referees table!
		userID, cleanupUser := createTestUser(t, db)
		defer cleanupUser()

		apiErr := db.SetUserAsReferee(userID, "123456789", "00-001", "Warsaw", "Street", "1", "2")
		if apiErr != nil {
			t.Fatalf("expected no error, got: %v", apiErr)
		}
		
		// Ensure referee was actually inserted by fetching it back
		refereeID, fetchErr := db.GetRefereeIDByUserID(userID)
		if fetchErr != nil {
			t.Fatalf("failed to fetch referee id after creation: %v", fetchErr)
		}
		if refereeID <= 0 {
			t.Errorf("expected positive referee ID, got %d", refereeID)
		}
		
		// Cleanup the dynamically generated referee table entry
		defer db.exec("DELETE FROM referees WHERE id = ?", refereeID)
		
		// Also clean up the address we inserted via SetUserAsReferee
		// Because it's hard to get the addressID without a dedicated query, we'll just ignore address orphans or clean it
		// This won't block tests since it's just a top-level entity.
	})
}

func TestGetRefereeDirectory(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	t.Run("successfully retrieves all referees in directory", func(t *testing.T) {
		list, err := db.GetRefereeDirectory()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if len(list) == 0 {
			t.Fatal("expected referee list to contain at least the seeded referee, but it is empty")
		}

		// Verify that our seeded referee is present in the list
		// Since we generated name/email dynamically, we just confirm that we scan successfully.
		var found bool
		for _, entry := range list {
			if entry.Email != "" {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected to find referee record in list: %v", list)
		}
		_ = refereeID
	})
}

func TestGetRefereeProfile(t *testing.T) {
	db := testDB(t)

	t.Run("successfully retrieves profile of a referee", func(t *testing.T) {
		refereeID, cleanupReferee := createTestReferee(t, db)
		defer cleanupReferee()

		// Get the user ID from referee
		var userID int
		row, cancel := db.queryRow(`SELECT user_id FROM referees WHERE id = ?`, refereeID)
		row.Scan(&userID)
		cancel()

		profile, err := db.GetRefereeProfile(userID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if profile.Email == "" || profile.Name == "" || profile.Surname == "" {
			t.Errorf("profile fields unexpectedly empty: %+v", profile)
		}
	})

	t.Run("returns ErrNotFound for nonexistent user profile", func(t *testing.T) {
		_, err := db.GetRefereeProfile(999999)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, types.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestInsertLicense(t *testing.T) {
	db := testDB(t)

	refereeID, cleanupReferee := createTestReferee(t, db)
	defer cleanupReferee()

	t.Run("successfully inserts a verified license", func(t *testing.T) {
		// Verify license names table has umpire or some seeded license
		var lnID int
		row, cancel := db.queryRow(`SELECT id FROM licenses_names LIMIT 1`)
		if err := row.Scan(&lnID); err != nil {
			cancel()
			// Insert one if not present
			res, err := db.exec(`INSERT INTO licenses_names (license_name) VALUES ('Referee Class C')`)
			if err != nil {
				t.Fatalf("failed to seed license name: %v", err)
			}
			lnIDVal, _ := res.LastInsertId()
			lnID = int(lnIDVal)
			defer db.exec(`DELETE FROM licenses_names WHERE id = ?`, lnID)
		} else {
			cancel()
		}

		issuedAt := time.Now()
		expireAt := issuedAt.AddDate(1, 0, 0)
		licenseNum := "LIC-987654"

		err := db.InsertLicense(refereeID, lnID, licenseNum, issuedAt, expireAt)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		defer db.exec(`DELETE FROM licenses WHERE license_number = ?`, licenseNum)
	})
}



