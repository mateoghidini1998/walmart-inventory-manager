package db

import (
	"database/sql"
	"fmt"
	"walmart-inventory-manager/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

func ConnectDB(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&tls=true", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Database connected")
	return db, nil
}
