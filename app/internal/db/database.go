package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr     string
	User     string
	Password string
}

func LoadConfig() (*Config, error) {
	config := new(Config)
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

func Open(config *Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/db?parseTime=true", config.User, config.Password, config.Addr)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}


