package db

import (
	"context"
	"testing"
	"tg-users-database/pkg/user"
)

var tests = []struct {
	username       string
	newStatus      string
	expectError    bool
	expectedStatus string
}{
	{"user1", "inactive", false, "inactive"},
	{"user2", "active", false, "active"},
	{"   ", "active", true, ""},
	{"user4", "active", false, "active"},
	{"", "active", true, ""},
	{"nonexistent", "active", false, "active"},
}

// testing functions uses db in RAM
var (
	ctx            = context.Background()
	dataSourceName = ":memory:"
	testDB, _      = NewDatabase(dataSourceName)
)

func TestNewDatabase(t *testing.T) {

	db, err := NewDatabase(dataSourceName)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v, wantErr %v", err, nil)
	}
	defer db.DB.Close()

	if db.DB == nil {
		t.Fatalf("Expected database to be initialized, but got nil")
	}

	var exists int
	err = db.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='users';").Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to query sqlite_master: %v", err)
	}
	if exists == 0 {
		t.Fatalf("Expected table 'users' to exist, but it does not")
	}
}

func TestCreateUser(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			usr := user.User{Username: tt.username, SubscriptionStatus: tt.newStatus}
			err := testDB.CreateUser(ctx, &usr)
			if (err != nil) != tt.expectError {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.expectError)
			}

			// Verify user was created
			if !tt.expectError {
				exists, err := testDB.UserExists(ctx, usr.Username)
				if err != nil {
					t.Errorf("Failed to check if user %s exists: %v", usr.Username, err)
				}
				if !exists {
					t.Errorf("User %s was not created as expected", usr.Username)
				}
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			user, err := testDB.GetUser(ctx, tt.username)

			if (err != nil) != tt.expectError {
				t.Errorf("GetUser() error = %v, wantErr %v", err, tt.expectError)
			}

			if !tt.expectError {
				if user == nil {
					t.Errorf("User %s not found", tt.username)
				}

				if user.Username != tt.username {
					t.Errorf("Expected username %s, got %s", tt.username, user.Username)
				}

				if user.SubscriptionStatus != tt.newStatus {
					t.Errorf("Expected subscription status %s, got %s", tt.newStatus, user.SubscriptionStatus)
				}
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			if err := testDB.UpdateUser(ctx, tt.username, tt.newStatus); (err != nil) != tt.expectError {
				t.Errorf("UpdateUser() error = %v, wantErr %v", err, tt.expectError)
			}

			if !tt.expectError {
				status, err := testDB.SubscriptionStatus(ctx, tt.username)
				if err != nil {
					t.Errorf("Failed to get subscription status: %v", err)
				}
				if status != tt.expectedStatus {
					t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
				}
			}
		})
	}
}

func TestSubscriptionStatus(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			status, err := testDB.SubscriptionStatus(ctx, tt.username)
			if (err != nil) != tt.expectError {
				t.Errorf("SubscriptionStatus() error = %v, wantErr %v", err, tt.expectError)
			}
			if status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
			}
		})
	}
}

func TestUserExists(t *testing.T) {
	tests := []struct {
		username       string
		expectedExists bool
	}{
		{"user1", true},
		{"user2", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			exists, err := testDB.UserExists(ctx, tt.username)
			if err != nil {
				t.Errorf("UserExists() error = %v", err)
			}
			if exists != tt.expectedExists {
				t.Errorf("Expected exists %v, got %v", tt.expectedExists, exists)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		username    string
		expectError bool
	}{
		{"user1", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			if err := testDB.DeleteUser(ctx, tt.username); (err != nil) != tt.expectError {
				t.Errorf("DeleteUser() error = %v, wantErr %v", err, tt.expectError)
			}

			// Check if the user was actually deleted
			if !tt.expectError {
				exists, err := testDB.UserExists(ctx, tt.username)
				if err != nil {
					t.Errorf("Failed to check if user exists: %v", err)
				}
				if exists {
					t.Errorf("User %s should have been deleted, but still exists", tt.username)
				}
			}
		})
	}
}
