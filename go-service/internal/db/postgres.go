package db

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenPostgres() (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		os.Getenv("PG_HOST"), os.Getenv("PG_PORT"), os.Getenv("PG_DB"),
		os.Getenv("PG_USER"), os.Getenv("PG_PASSWORD"))
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	maxOpen := envInt("PG_POOL_MAX", 40)
	database.SetMaxOpenConns(maxOpen)
	database.SetMaxIdleConns(maxOpen / 2)
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
