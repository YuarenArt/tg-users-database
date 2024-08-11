package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"tg-users-database/pkg/db"
	"time"

	"github.com/stretchr/testify/assert"
)

// TODO creates func for generating test cases

var dataSourceName = ":memory:"

var testCases = []struct {
	name               string
	initialUser        db.User
	updateUser         db.User
	method             string
	url                string
	body               interface{}
	expectedStatusCode int
	expectedResponse   interface{}
}{
	{
		name:   "CreateUser",
		method: http.MethodPost,
		url:    "/users",
		body: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
		expectedStatusCode: http.StatusCreated,
		expectedResponse: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
	},
	{
		name: "GetUser",
		initialUser: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
		method:             http.MethodGet,
		url:                "/users/testuser",
		expectedStatusCode: http.StatusOK,
		expectedResponse: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
	},
	{
		name: "UpdateUserSubscription",
		initialUser: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
		method: http.MethodPut,
		url:    "/users/testuser",
		body: db.User{
			Subscription: db.Subscription{
				SubscriptionStatus: "inactive",
				Duration:           "2 months",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 2, 0),
			},
		},
		expectedStatusCode: http.StatusOK,
		expectedResponse: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "inactive",
				Duration:           "2 months",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 2, 0),
			},
		},
	},
	{
		name: "DeleteUser",
		initialUser: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
		method:             http.MethodDelete,
		url:                "/users/testuser",
		expectedStatusCode: http.StatusNoContent,
	},
	{
		name: "SubscriptionStatus",
		initialUser: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
		method:             http.MethodGet,
		url:                "/users/testuser/subscription",
		expectedStatusCode: http.StatusOK,
		expectedResponse:   "active",
	},
	{
		name: "IsUserExists",
		initialUser: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
		method:             http.MethodGet,
		url:                "/users/testuser/exists",
		expectedStatusCode: http.StatusOK,
		expectedResponse:   true,
	},
	{
		name: "UpdateUserTraffic",
		initialUser: db.User{
			Username: "testuser",
			ChatID:   12345,
			Subscription: db.Subscription{
				SubscriptionStatus: "active",
				Duration:           "1 month",
				StartSubscription:  time.Now(),
				EndSubscription:    time.Now().AddDate(0, 1, 0),
			},
		},
		method:             http.MethodPut,
		url:                "/users/testuser/traffic",
		body:               100.0,
		expectedStatusCode: http.StatusOK,
		expectedResponse: map[string]string{
			"message": "Traffic updated successfully",
		},
	},
}

// Setup test environment
func setupTestEnvironment() (*UserHandler, *db.Database) {
	db, err := db.NewDatabase(dataSourceName)
	if err != nil {
		panic(err)
	}

	handler := NewHandler(db)
	return handler, db
}

func TestHandlers(t *testing.T) {
	for _, tc := range testCases {
		tc := tc // capture the range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run tests in parallel

			// Setup test environment
			h, db := setupTestEnvironment()
			defer db.DB.Close()

			// Setup initial state
			if tc.initialUser.Username != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := db.CreateUser(ctx, &tc.initialUser); err != nil {
					t.Fatalf("Failed to create initial user: %v", err)
				}
			}

			var body *bytes.Buffer
			if tc.body != nil {
				bodyBytes, _ := json.Marshal(tc.body)
				body = bytes.NewBuffer(bodyBytes)
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tc.method, tc.url, body)
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			h.Router.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatusCode, rec.Code)

			if tc.expectedResponse != nil {
				var actualResponse map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &actualResponse)
				if err != nil {
					t.Fatalf("Failed to parse response body: %v", err)
				}

				expectedBytes, _ := json.Marshal(tc.expectedResponse)
				actualBytes, _ := json.Marshal(actualResponse)
				assert.JSONEq(t, string(expectedBytes), string(actualBytes))
			}
		})
	}
}
