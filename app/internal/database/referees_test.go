package database

import (
	"testing"
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
