package db

import (
	"context"
	"log"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ctx            = context.Background()
	dataSourceName = ":memory:"
)

func setupTestDB() (*Database, error) {
	db, err := NewDatabase(dataSourceName)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func teardownTestDB(db *Database) {
	db.DB.Close()
}

// Test functions
func TestCreateUser(t *testing.T) {
	type testCase struct {
		name       string
		user       User
		wantErr    bool
		errMessage string
	}

	testCases := []testCase{
		{
			name: "ValidUser",
			user: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			wantErr: false,
		},
		{
			name: "EmptyUsername",
			user: User{
				Username: "",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			wantErr:    true,
			errMessage: "unsupported username",
		},
		{
			name: "DuplicateUser",
			user: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			wantErr:    true,
			errMessage: "failed to execute insert statement: UNIQUE constraint failed: users.username",
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := db.CreateUser(ctx, &tc.user)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}
			if tc.wantErr && err != nil && err.Error() != tc.errMessage {
				t.Fatalf("Expected error message: %s, got: %s", tc.errMessage, err.Error())
			}
		})
	}
}

func TestUpdateUserSubscription(t *testing.T) {
	type testCase struct {
		name            string
		initialUser     User
		newSubscription Subscription
		wantErr         bool
		errMessage      string
	}

	testCases := []testCase{
		{
			name: "ValidUpdate",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			newSubscription: Subscription{
				SubscriptionStatus: "inactive",
				Duration:           "2 months",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 2, 0),
			},
			wantErr: false,
		},
		{
			name: "UserDoesNotExist",
			initialUser: User{
				Username: "nonexistentuser",
				ChatID:   12345,
			},
			newSubscription: Subscription{
				SubscriptionStatus: "inactive",
				Duration:           "2 months",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 2, 0),
			},
			wantErr:    true,
			errMessage: "user nonexistentuser not found",
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log.Printf("Running test case: %s", tc.name)

			if tc.name != "UserDoesNotExist" {
				log.Printf("Creating initial user: %v", tc.initialUser)
				err := db.CreateUser(ctx, &tc.initialUser)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			log.Printf("Updating subscription for user: %s", tc.initialUser.Username)
			err = db.UpdateUserSubscription(ctx, tc.initialUser.Username, tc.newSubscription)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}
			if tc.wantErr && err != nil && err.Error() != tc.errMessage {
				t.Fatalf("Expected error message: %s, got: %s", tc.errMessage, err.Error())
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	type testCase struct {
		name        string
		initialUser User
		wantErr     bool
		errMessage  string
	}

	testCases := []testCase{
		{
			name: "ValidDelete",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			wantErr: false,
		},
		{
			name: "UserDoesNotExist",
			initialUser: User{
				Username: "nonexistentuser",
				ChatID:   12345,
			},
			wantErr:    true,
			errMessage: "user nonexistentuser not found",
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "UserDoesNotExist" {
				err := db.CreateUser(ctx, &tc.initialUser)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			err = db.DeleteUser(ctx, tc.initialUser.Username)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}
			if tc.wantErr && err != nil && err.Error() != tc.errMessage {
				t.Fatalf("Expected error message: %s, got: %s", tc.errMessage, err.Error())
			}
		})
	}
}

func TestIsUserExists(t *testing.T) {
	type testCase struct {
		name        string
		initialUser User
		username    string
		wantExists  bool
		wantErr     bool
	}

	testCases := []testCase{
		{
			name: "UserExists",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username:   "testuser",
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "UserDoesNotExist",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username:   "nonexistentuser",
			wantExists: false,
			wantErr:    false,
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "UserDoesNotExist" {
				err := db.CreateUser(ctx, &tc.initialUser)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			exists, err := db.IsUserExists(ctx, tc.username)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}
			if exists != tc.wantExists {
				t.Fatalf("Expected user exists: %v, got: %v", tc.wantExists, exists)
			}
		})
	}
}

func TestUser(t *testing.T) {
	type testCase struct {
		name        string
		initialUser User
		username    string
		wantUser    *User
		wantErr     bool
	}

	testCases := []testCase{
		{
			name: "ValidUser",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username: "testuser",
			wantUser: &User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			wantErr: false,
		},
		{
			name: "UserDoesNotExist",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username: "nonexistentuser",
			wantUser: nil,
			wantErr:  true,
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "UserDoesNotExist" {
				err := db.CreateUser(ctx, &tc.initialUser)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			user, err := db.User(ctx, tc.username)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}
			if tc.wantUser != nil {
				if user.Username != tc.wantUser.Username || user.ChatID != tc.wantUser.ChatID || user.Subscription.SubscriptionStatus != tc.wantUser.Subscription.SubscriptionStatus {
					t.Fatalf("Expected user: %v, got: %v", tc.wantUser, user)
				}
			} else {
				if user != nil {
					t.Fatalf("Expected user to be nil, got: %v", user)
				}
			}
		})
	}
}

func TestSubscriptionStatus(t *testing.T) {
	type testCase struct {
		name        string
		initialUser User
		username    string
		wantStatus  string
		wantErr     bool
	}

	testCases := []testCase{
		{
			name: "ValidSubscriptionStatus",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username:   "testuser",
			wantStatus: "active",
			wantErr:    false,
		},
		{
			name: "UserDoesNotExist",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username:   "nonexistentuser",
			wantStatus: "",
			wantErr:    true,
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "UserDoesNotExist" {
				err := db.CreateUser(ctx, &tc.initialUser)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			status, err := db.SubscriptionStatus(ctx, tc.username)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}
			if status != tc.wantStatus {
				t.Fatalf("Expected status: %s, got: %s", tc.wantStatus, status)
			}
		})
	}
}

func TestUpdateUserTraffic(t *testing.T) {
	type testCase struct {
		name        string
		initialUser User
		username    string
		traffic     float64
		wantErr     bool
	}

	testCases := []testCase{
		{
			name: "ValidUpdateTraffic",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username: "testuser",
			traffic:  100.0,
			wantErr:  false,
		},
		{
			name: "UserDoesNotExist",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username: "nonexistentuser",
			traffic:  100.0,
			wantErr:  false, // No error expected for non-existent user
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "UserDoesNotExist" {
				err := db.CreateUser(ctx, &tc.initialUser)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			err = db.UpdateUserTraffic(ctx, tc.username, tc.traffic)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}

			if !tc.wantErr {
				user, err := db.User(ctx, tc.username)
				if tc.name == "UserDoesNotExist" {
					if err == nil {
						t.Fatalf("Expected error for non-existent user, got: %v", err)
					}
				} else {
					if err != nil {
						t.Fatalf("Failed to retrieve user: %v", err)
					}
					if user.Traffic != tc.traffic {
						t.Fatalf("Expected traffic: %f, got: %f", tc.traffic, user.Traffic)
					}
				}
			}
		})
	}
}

func TestResetUserTraffic(t *testing.T) {
	type testCase struct {
		name        string
		initialUser User
		username    string
		wantErr     bool
	}

	testCases := []testCase{
		{
			name: "ValidResetTraffic",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
				Traffic: 100.0,
			},
			username: "testuser",
			wantErr:  false,
		},
		{
			name: "UserDoesNotExist",
			initialUser: User{
				Username: "testuser",
				ChatID:   12345,
				Subscription: Subscription{
					SubscriptionStatus: "active",
					Duration:           "1 month",
					StartSubscription:  time.Now(),
					EndSubscription:    time.Now().AddDate(0, 1, 0),
				},
			},
			username: "nonexistentuser",
			wantErr:  false, // No error expected for non-existent user
		},
	}

	db, err := setupTestDB()
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer teardownTestDB(db)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name != "UserDoesNotExist" {
				err := db.CreateUser(ctx, &tc.initialUser)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			err = db.ResetUserTraffic(ctx, tc.username)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}

			if !tc.wantErr {
				user, err := db.User(ctx, tc.username)
				if tc.name == "UserDoesNotExist" {
					if err == nil {
						t.Fatalf("Expected error for non-existent user, got: %v", err)
					}
				} else {
					if err != nil {
						t.Fatalf("Failed to retrieve user: %v", err)
					}
					if user.Traffic != 0.0 {
						t.Fatalf("Expected traffic: 0.0, got: %f", user.Traffic)
					}
				}
			}
		})
	}
}

func TestAllUsername(t *testing.T) {
	type testCase struct {
		name          string
		initialUsers  []User
		wantUsernames []string
		wantErr       bool
	}

	testCases := []testCase{
		{
			name: "ValidAllUsername",
			initialUsers: []User{
				{
					Username: "testuser1",
					ChatID:   12345,
					Subscription: Subscription{
						SubscriptionStatus: "active",
						Duration:           "1 month",
						StartSubscription:  time.Now(),
						EndSubscription:    time.Now().AddDate(0, 1, 0),
					},
				},
				{
					Username: "testuser2",
					ChatID:   67890,
					Subscription: Subscription{
						SubscriptionStatus: "active",
						Duration:           "1 month",
						StartSubscription:  time.Now(),
						EndSubscription:    time.Now().AddDate(0, 1, 0),
					},
				},
			},
			wantUsernames: []string{"testuser1", "testuser2"},
			wantErr:       false,
		},
		{
			name:          "NoUsers",
			initialUsers:  []User{},
			wantUsernames: []string{},
			wantErr:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := setupTestDB()
			if err != nil {
				t.Fatalf("Failed to setup test database: %v", err)
			}
			defer teardownTestDB(db)

			for _, user := range tc.initialUsers {
				err := db.CreateUser(ctx, &user)
				if err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			usernames, err := db.AllUsername(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Expected error: %v, got: %v", tc.wantErr, err)
			}
			if len(usernames) != len(tc.wantUsernames) {
				t.Fatalf("Expected usernames length: %d, got: %d", len(tc.wantUsernames), len(usernames))
			}
			for _, username := range tc.wantUsernames {
				if !contains(usernames, username) {
					t.Fatalf("Expected username: %s, not found in usernames", username)
				}
			}
		})
	}
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
