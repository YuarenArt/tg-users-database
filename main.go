package main

import (
	"log"
	"tg-users-database/pkg/db"
	"tg-users-database/pkg/handler"
)

// TODO add fucntion for recording traffic from user

// @title User Database API
// @version 1.0
// @description This is a sample server for managing user subscriptions.

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8082
// @BasePath /
func main() {
	// Initialize the database connection
	database, err := db.NewDatabase("users.db")
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	// Initialize the handler with the database
	handler := handler.NewHandler(database)
	if err := handler.Router.Run(":8082"); err != nil {
		log.Fatalf("Failed to start the server: %v", err)
	}
}
