package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	Username     string       `json:"username"`
	Subscription Subscription `json:"subscription"`
	Traffic      float64      `json:"traffic"`
	ChatID       int64        `json:"chat_id"`
}

type Subscription struct {
	ID                 int64     `json:"id"`
	SubscriptionStatus string    `json:"subscription_status"` // active, inactive
	Duration           string    `json:"duration"`            // month, year, forever
	StartSubscription  time.Time `json:"start_subscription"`
	EndSubscription    time.Time `json:"end_subscription"`
}

type Database struct {
	DB *sql.DB
	mu sync.Mutex
}

// SQL Queries
const (
	createTableUsers = `
    CREATE TABLE IF NOT EXISTS users (
        username TEXT PRIMARY KEY,
        subscription_id INTEGER NOT NULL,
        traffic REAL DEFAULT 0,
        chat_id INTEGER,
        FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
    );`

	createTableSubscriptions = `
    CREATE TABLE IF NOT EXISTS subscriptions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        subscription_status TEXT DEFAULT "inactive",
        duration TEXT NOT NULL,
        start_subscription DATE NOT NULL,
        end_subscription DATE NOT NULL
    );`

	selectUserSQL = `
    		SELECT  users.username, users.traffic, users.chat_id, 
           			subscriptions.id, subscriptions.subscription_status, 
          			subscriptions.duration, subscriptions.start_subscription, subscriptions.end_subscription
    		FROM users 
    		JOIN subscriptions ON users.subscription_id = subscriptions.id 
    		WHERE users.username = ?`

	updateUserSubscriptionSQL = `
    		UPDATE subscriptions 
   	 		SET subscription_status = ?, duration = ?, start_subscription = ?, end_subscription = ?
    		WHERE id = (SELECT subscription_id FROM users WHERE username = ?)`

	userSubscriptionStatusSQL = `
								SELECT subscriptions.subscription_status 
								FROM users 
								JOIN subscriptions ON users.subscription_id = subscriptions.id 
								WHERE users.username = ?`

	deleteSubscriptionIfUnusedSQL = `
    		DELETE FROM subscriptions 
    		WHERE id = ? AND NOT EXISTS (
       	    SELECT 1 FROM users WHERE subscription_id = ?)`

	insertUserSQL        = "INSERT INTO users (username, subscription_id, chat_id) VALUES (?, ?, ?)"
	deleteUserSQL        = "DELETE FROM users WHERE username = ?"
	userExistsSQL        = "SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)"
	addSubscription      = "INSERT INTO subscriptions (subscription_status, duration, start_subscription, end_subscription) VALUES (?, ?, ?, ?)"
	subscriptionId       = "SELECT subscription_id FROM users WHERE username = ?"
	updateUserTrafficSQL = "UPDATE users SET traffic = ? WHERE username = ?"
	allUsername          = "SELECT username FROM users"
)

const timeFormat = "2006-01-02T15:04:05Z"

var dbInitMu sync.Mutex

// NewDatabase initializes and returns a new Database instance
func NewDatabase(dataSourceName string) (*Database, error) {
	dbInitMu.Lock()
	defer dbInitMu.Unlock()

	log.Println("Opening database connection...")
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Initialize subscriptions table
	_, err = db.Exec(createTableSubscriptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions table: %w", err)
	}

	// Initialize users table
	_, err = db.Exec(createTableUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}
	log.Println("Database connection established successfully.")

	return &Database{
		DB: db,
	}, nil
}

func FormatTime(t time.Time) string {
	return t.Format(timeFormat)
}

// addSubscription inserts a new subscription into the subscriptions table
func (db *Database) addSubscription(ctx context.Context, subscription *Subscription) (int64, error) {
	stmt, err := db.DB.PrepareContext(ctx, addSubscription)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare subscription insert statement: %w", err)
	}
	defer stmt.Close()

	startSubscription := FormatTime(subscription.StartSubscription)
	endSubscription := FormatTime(subscription.EndSubscription)

	res, err := stmt.ExecContext(ctx, subscription.SubscriptionStatus, subscription.Duration, startSubscription, endSubscription)
	if err != nil {
		return 0, fmt.Errorf("failed to execute subscription insert statement: %w", err)
	}
	return res.LastInsertId()
}

// CreateUser adds a new user to the database
func (db *Database) CreateUser(ctx context.Context, user *User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	log.Printf("Preparing to insert user: %s", user.Username)

	if strings.TrimSpace(user.Username) == "" {
		return errors.New("unsupported username")
	}

	subscriptionID, err := db.addSubscription(ctx, &user.Subscription)
	if err != nil {
		return fmt.Errorf("failed to add subscription: %w", err)
	}

	stmt, err := db.DB.PrepareContext(ctx, insertUserSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, user.Username, subscriptionID, user.ChatID)
	if err != nil {
		return fmt.Errorf("failed to execute insert statement: %w", err)
	}

	log.Printf("User %s created successfully.", user.Username)
	return nil
}

// User retrieves a user by Telegram username
func (db *Database) User(ctx context.Context, username string) (*User, error) {

	log.Printf("Retrieving user: %s", username)
	var usr User
	var sub Subscription

	row := db.DB.QueryRowContext(ctx, selectUserSQL, username)

	var startSubscription, endSubscription string

	err := row.Scan(
		&usr.Username,
		&usr.Traffic,
		&usr.ChatID,
		&sub.ID,
		&sub.SubscriptionStatus,
		&sub.Duration,
		&startSubscription,
		&endSubscription,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("User %s not found.", username)
			return nil, err
		}
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	sub.StartSubscription, err = time.Parse(timeFormat, startSubscription)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start_subscription: %w", err)
	}

	sub.EndSubscription, err = time.Parse(timeFormat, endSubscription)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end_subscription: %w", err)
	}

	usr.Subscription = sub
	log.Printf("User retrieved: %s", username)
	return &usr, nil
}

// UpdateUserSubscription updates a user's subscription status
func (db *Database) UpdateUserSubscription(ctx context.Context, username string, newSubscription Subscription) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	log.Printf("Updating user: %s", username)

	exists, err := db.IsUserExists(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	stmt, err := db.DB.PrepareContext(ctx, updateUserSubscriptionSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	startSubscription := FormatTime(newSubscription.StartSubscription)
	endSubscription := FormatTime(newSubscription.EndSubscription)

	_, err = stmt.ExecContext(ctx, newSubscription.SubscriptionStatus, newSubscription.Duration, startSubscription, endSubscription, username)
	if err != nil {
		return fmt.Errorf("failed to execute update statement: %w", err)
	}

	log.Printf("User %s updated successfully.", username)
	return nil
}

// DeleteUser removes a user from the database
func (db *Database) DeleteUser(ctx context.Context, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	log.Printf("Preparing to delete user: %s", username)

	var subscriptionID int64
	err := db.DB.QueryRowContext(ctx, "SELECT subscription_id FROM users WHERE username = ?", username).Scan(&subscriptionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user %s not found", username)
		}
		return fmt.Errorf("failed to get subscription ID: %w", err)
	}

	stmt, err := db.DB.PrepareContext(ctx, deleteUserSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to execute delete statement: %w", err)
	}

	stmt, err = db.DB.PrepareContext(ctx, deleteSubscriptionIfUnusedSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare delete subscription statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, subscriptionID, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to execute delete subscription statement: %w", err)
	}

	log.Printf("User %s and potentially their subscription deleted successfully.", username)
	return nil
}

// IsUserExists checks if a user exists in the database
func (db *Database) IsUserExists(ctx context.Context, username string) (bool, error) {

	log.Printf("Checking if user exists: %s", username)
	var exists bool
	err := db.DB.QueryRowContext(ctx, userExistsSQL, username).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}

	log.Printf("User: %s, exists: %v", username, exists)
	return exists, nil
}

// SubscriptionStatus returns the user's subscription status
func (db *Database) SubscriptionStatus(ctx context.Context, username string) (string, error) {

	log.Printf("Checking subscription status: %s", username)

	var subscriptionStatus string
	err := db.DB.QueryRowContext(ctx, userSubscriptionStatusSQL, username).Scan(&subscriptionStatus)
	if err != nil {
		return "", fmt.Errorf("failed to check subscription status: %w", err)
	}
	log.Printf("User: %s, status: %v", username, subscriptionStatus)
	return subscriptionStatus, nil
}

// UpdateUserTraffic changes the user's traffic value
func (db *Database) UpdateUserTraffic(ctx context.Context, username string, traffic float64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	log.Printf("Updating traffic for user: %s", username)

	stmt, err := db.DB.PrepareContext(ctx, updateUserTrafficSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, traffic, username)
	if err != nil {
		return fmt.Errorf("failed to execute update statement: %w", err)
	}

	log.Printf("Traffic for user %s updated successfully.", username)
	return nil
}

// ResetUserTraffic resets the traffic for a user
func (db *Database) ResetUserTraffic(ctx context.Context, username string) error {
	return db.UpdateUserTraffic(ctx, username, 0)
}

// AllUsername return all username
func (db *Database) AllUsername(ctx context.Context) ([]string, error) {
	rows, err := db.DB.QueryContext(ctx, allUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		usernames = append(usernames, username)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return usernames, nil
}
