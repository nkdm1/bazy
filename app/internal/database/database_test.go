package database

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// testDB initializes a clean database connection for testing.
// Tests should use factories_test.go to set up their own specific data.
func testDB(t *testing.T) *Database {
	t.Helper()

	db := Init()
	t.Cleanup(func() {
		db.instance.Close()
	})

	return db
}
