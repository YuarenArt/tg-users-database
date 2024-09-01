package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
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
        subscription_id SERIAL NOT NULL,
        traffic REAL DEFAULT 0,
        chat_id BIGINT,
        FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
    );`

	createTableSubscriptions = `
    CREATE TABLE IF NOT EXISTS subscriptions (
        id SERIAL PRIMARY KEY,
        subscription_status TEXT DEFAULT 'inactive',
        duration TEXT NOT NULL DEFAULT 'month',
        start_subscription TIMESTAMP NOT NULL,
        end_subscription TIMESTAMP NOT NULL
    );`

	selectUserSQL = `
    		SELECT  users.username, users.traffic, users.chat_id, 
           			subscriptions.id, subscriptions.subscription_status, 
          			subscriptions.duration, subscriptions.start_subscription, subscriptions.end_subscription
    		FROM users 
    		JOIN subscriptions ON users.subscription_id = subscriptions.id 
    		WHERE users.username = $1`

	updateUserSubscriptionSQL = `
    		UPDATE subscriptions 
        	SET subscription_status = $1, duration = $2, start_subscription = $3, end_subscription = $4
        	WHERE id = (SELECT subscription_id FROM users WHERE username = $5)`

	userSubscriptionStatusSQL = `
			SELECT subscriptions.subscription_status 
			FROM users 
			JOIN subscriptions ON users.subscription_id = subscriptions.id 
			WHERE users.username = $1`

	deleteSubscriptionIfUnusedSQL = `
            DELETE FROM subscriptions 
            WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM users WHERE subscription_id = $1)`
	unusedSubscriptionsSQL = `
            SELECT id FROM subscriptions 
            WHERE NOT EXISTS (SELECT 1 FROM users WHERE users.subscription_id = subscriptions.id)`

	insertUserSQL        = "INSERT INTO users (username, subscription_id, chat_id) VALUES ($1, $2, $3)"
	deleteUserSQL        = "DELETE FROM users WHERE username = $1"
	userExistsSQL        = "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)"
	addSubscription      = "INSERT INTO subscriptions (subscription_status, duration, start_subscription, end_subscription) VALUES ($1, $2, $3, $4) RETURNING id"
	subscriptionId       = "SELECT subscription_id FROM users WHERE username = $1"
	updateUserTrafficSQL = "UPDATE users SET traffic = $1 WHERE username = $2"
	allUsername          = "SELECT username FROM users"
)

const timeFormat = time.RFC3339

func FormatTime(t time.Time) string {
	return t.Format(timeFormat)
}

var dbInitMu sync.Mutex

/*
var (
	pgInstance *Database
	pgOnce     sync.Once
)
*/
// NewDatabase initializes and returns a new Database instance
func NewDatabase(dataSourceName string) (*Database, error) {
	dbInitMu.Lock()
	defer dbInitMu.Unlock()

	log.Println("Opening database connection...")

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	defaultConnStr := fmt.Sprintf(
		"user=%s password=%s dbname=postgres host=%s port=%s sslmode=%s",
		user, password, host, port, sslmode,
	)

	log.Println("defaultConnStr: ", defaultConnStr)

	defaultDB, err := sql.Open("postgres", defaultConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open default database: %w", err)
	}
	defer defaultDB.Close()

	// Create the new database
	_, err = defaultDB.Exec("CREATE DATABASE users")
	if err != nil && err.Error() != "pq: database \"users\" already exists" {
		log.Printf("failed to create database: %s", err.Error())
	}

	// Connect to the newly created database
	ConnStr := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=%s",
		user, password, dbname, host, port, sslmode,
	)
	db, err := sql.Open("postgres", ConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the new database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(time.Hour)

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

	// Create a new Database instance
	newDB := &Database{
		DB: db,
	}

	// Clean up unused subscriptions
	err = newDB.cleanupUnusedSubscriptions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to clean up unused subscriptions: %w", err)
	}

	log.Println("Database connection established successfully.")

	return newDB, nil
}

// cleanupUnusedSubscriptions deletes all unused subscriptions
func (db *Database) cleanupUnusedSubscriptions(ctx context.Context) error {
	rows, err := db.DB.QueryContext(ctx, unusedSubscriptionsSQL)
	if err != nil {
		return fmt.Errorf("failed to execute unused subscriptions query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var subscriptionID int64
		if err := rows.Scan(&subscriptionID); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		stmt, err := db.DB.PrepareContext(ctx, deleteSubscriptionIfUnusedSQL)
		if err != nil {
			return fmt.Errorf("failed to prepare delete subscription statement: %w", err)
		}
		defer stmt.Close()

		_, err = stmt.ExecContext(ctx, subscriptionID)
		if err != nil {
			return fmt.Errorf("failed to execute delete subscription statement: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration error: %w", err)
	}

	return nil
}

// addSubscription inserts a new empty subscription into the subscriptions table
func (db *Database) addSubscription(ctx context.Context) (int64, error) {
	stmt, err := db.DB.PrepareContext(ctx, addSubscription)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare subscription insert statement: %w", err)
	}
	defer stmt.Close()

	startSubscription := FormatTime(time.Now())
	endSubscription := FormatTime(time.Time{})
	duration := "month"
	suscriptionStatus := "inactive"

	var subscriptionID int64
	err = stmt.QueryRowContext(ctx, suscriptionStatus, duration, startSubscription, endSubscription).Scan(&subscriptionID)
	if err != nil {
		return 0, fmt.Errorf("failed to execute subscription insert statement: %w", err)
	}
	return subscriptionID, nil
}

// CreateUser adds a new user to the database
func (db *Database) CreateUser(ctx context.Context, user *User) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	log.Printf("Preparing to insert user: %s", user.Username)

	if strings.TrimSpace(user.Username) == "" {
		return errors.New("unsupported username")
	}

	subscriptionID, err := db.addSubscription(ctx)
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

	stmt, err := db.DB.PrepareContext(ctx, deleteUserSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to execute delete statement: %w", err)
	}

	log.Printf("User %s and their subscription deleted successfully.", username)
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
