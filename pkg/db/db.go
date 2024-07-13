package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"tg-users-database/pkg/user"
)

type Database struct {
	DB *sql.DB
}

// SQL Queries
const (
	createTableSQL = `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL UNIQUE,
        subscription_status TEXT NOT NULL
    );`

	insertUserSQL             = "INSERT INTO users (username, subscription_status) VALUES (?, ?)"
	selectUserSQL             = "SELECT id, username, subscription_status FROM users WHERE username = ?"
	updateUserSQL             = "UPDATE users SET subscription_status = ? WHERE username = ?"
	deleteUserSQL             = "DELETE FROM users WHERE username = ?"
	userExistsSQL             = "SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)"
	userSabscriptionStatusSQL = "SELECT subscription_status FROM users WHERE username = ?"
)

// NewDatabase creates a connection with the database
func NewDatabase(dataSourceName string) (*Database, error) {
	log.Println("Opening database connection...")
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize users' table
	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	log.Println("Database connection established successfully.")
	return &Database{DB: db}, nil
}

// CreateUser adds a new user to the database
func (db *Database) CreateUser(ctx context.Context, usr *user.User) error {
	log.Printf("Preparing to insert user: %s", usr.TelegramUsername)
	stmt, err := db.DB.PrepareContext(ctx, insertUserSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, usr.TelegramUsername, usr.SubscriptionStatus)
	if err != nil {
		return fmt.Errorf("failed to execute insert statement: %w", err)
	}

	log.Printf("User %s created successfully.", usr.TelegramUsername)
	return nil
}

// GetUser retrieves a user by Telegram username
func (db *Database) GetUser(ctx context.Context, username string) (*user.User, error) {
	log.Printf("Retrieving user: %s", username)
	var usr user.User
	row := db.DB.QueryRowContext(ctx, selectUserSQL, username)

	err := row.Scan(&usr.ID, &usr.TelegramUsername, &usr.SubscriptionStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("User %s not found.", username)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	log.Printf("User retrieved: %s", username)
	return &usr, nil
}

// UpdateUser updates a user's subscription status
func (db *Database) UpdateUser(ctx context.Context, username string, subscriptionStatus string) error {
	log.Printf("Updating user: %s", username)

	exists, err := db.UserExists(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	stmt, err := db.DB.PrepareContext(ctx, updateUserSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, subscriptionStatus, username)
	if err != nil {
		return fmt.Errorf("failed to execute update statement: %w", err)
	}

	log.Printf("User %s updated successfully.", username)
	return nil
}

// DeleteUser removes a user from the database
func (db *Database) DeleteUser(ctx context.Context, username string) error {
	log.Printf("Preparing to delete user: %s", username)
	stmt, err := db.DB.PrepareContext(ctx, deleteUserSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to execute delete statement: %w", err)
	}

	log.Printf("User %s deleted successfully.", username)
	return nil
}

// UserExists checks if a user exists in the database
func (db *Database) UserExists(ctx context.Context, username string) (bool, error) {
	log.Printf("Checking if user exists: %s", username)
	var exists bool
	err := db.DB.QueryRowContext(ctx, userExistsSQL, username).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}

	log.Printf("User: %s, exists: %v", username, exists)
	return exists, nil
}

func (db *Database) SubscriptionStatus(ctx context.Context, username string) (string, error) {
	log.Printf("Checking subscription status: %s", username)

	var subscriptionStatus string
	err := db.DB.QueryRowContext(ctx, userSabscriptionStatusSQL, username).Scan(&subscriptionStatus)
	if err != nil {
		return "", fmt.Errorf("failed to check if user exists: %w", err)
	}
	log.Printf("User: %s, status: %v", username, subscriptionStatus)
	return subscriptionStatus, nil
}
