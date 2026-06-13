package main
import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)
func main() {
	db, err := sql.Open("mysql", "root:root@tcp(ubuntu:3306)/db")
	if err != nil { panic(err) }
	
	rows, _ := db.Query("SELECT referee_id, license_number, license_name_id, issued_at, expire_at FROM licenses")
	for rows.Next() {
		var id, nid int
		var num, ia, ea string
		rows.Scan(&id, &num, &nid, &ia, &ea)
		fmt.Printf("License: ref_id=%d, num=%s, name_id=%d, issued_at=%s, expire_at=%s\n", id, num, nid, ia, ea)
	}

	rows2, _ := db.Query("SELECT id, user_id FROM referees")
	for rows2.Next() {
		var id, uid int
		rows2.Scan(&id, &uid)
		fmt.Printf("Referee: id=%d, user_id=%d\n", id, uid)
	}
}
