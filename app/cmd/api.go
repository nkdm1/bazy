package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nkdm1/bazy/internal/database"
	"github.com/nkdm1/bazy/internal/handlers"

	_ "github.com/go-sql-driver/mysql"
)

func (app *api) mount() http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.Timeout(10*time.Second),
	)
	r.Get("/status", handlers.GetStatus)
	return r
}

func (app *api) run(h http.Handler) {
	log.Println("server up on :8080")
	if err := http.ListenAndServe(":8080", h); err != nil {
		panic(err)
	}
}

func databaseConnect() *sql.DB {
	config, err := database.LoadConfig()
	if err != nil {
		panic(err)
	}
	log.Printf("connecting to the database on %s\n", config.Addr)
	dbInstance, err := database.Connect(config)
	if err != nil {
		panic(err)
	}
	log.Println("successfully connected to the database")
	return dbInstance
}

type api struct {
	db *sql.DB
}
