package main

import (
	"fmt"
	"log"

	"app/internal/db"
	"app/internal/queries"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	config, err := db.LoadConfig()
	if err != nil {
		panic(err)
	}
	dbInstance, err := db.Open(config)
	if err != nil {
		panic(err)
	}
	log.Printf("successfully connected to %s\n", config.Addr)
	defer dbInstance.Close()

	tables, err := queries.GetTables(dbInstance)
	if err != nil {
		panic(err)
	}

	fmt.Println("tables")
	for _, table := range tables {
		fmt.Printf("\t%s\n", table)
	}
}
