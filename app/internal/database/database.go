package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Database struct {
	instance *sql.DB
	timeout  time.Duration
}

func (db *Database) query(query string, args ...any) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	defer cancel()

	return db.instance.QueryContext(ctx, query, args...)
}

func (db *Database) queryRow(query string, args ...any) *sql.Row {
	ctx, cancel := context.WithTimeout(context.Background(), db.timeout)
	defer cancel()

	return db.instance.QueryRowContext(ctx, query, args...)
}

type config struct {
	Addr     string
	User     string
	Password string
}

func loadConfig() (*config, error) {
	config := new(config)
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf(".env file not found")
	}
	config.Addr = os.Getenv("DB_ADDR")
	if config.Addr == "" {
		return nil, fmt.Errorf("\"DB_ADDR\" variable is not set in .env file")
	}
	config.User = os.Getenv("DB_USER")
	if config.User == "" {
		return nil, fmt.Errorf("\"DB_USER\" variable is not set in .env file")
	}
	config.Password = os.Getenv("DB_PASSWORD")
	if config.Password == "" {
		return nil, fmt.Errorf("\"DB_PASSWORD\" variable is not set in .env file")
	}
	return config, nil
}

func connect(config *config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/db?parseTime=true", config.User, config.Password, config.Addr)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func Init() *Database {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}
	log.Printf("connecting to the database on %s\n", config.Addr)
	instance, err := connect(config)
	if err != nil {
		panic(err)
	}
	log.Println("successfully connected to the database")
	return &Database{
		instance,
		time.Second * 5,
	}
}
