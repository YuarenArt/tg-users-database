package main

import (
	"log"

	"github.com/YuarenArt/tg-users-database/pkg/db"
	"github.com/YuarenArt/tg-users-database/pkg/handler"
	"github.com/YuarenArt/tg-users-database/pkg/scheduler"
)

// @title user Database API
// @version 2.2
// @description This is a server for managing user subscriptions over HTTPS.

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization

// @host localhost:8082
// @BasePath /
// @schemes https
func main() {
	// Initialize the database connection
	database, err := db.NewDatabase("users.db")
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	scheduler := scheduler.NewScheduler(database)
	scheduler.Start()

	certFile := "cert.pem"
	keyFile := "key.pem"

	// Initialize the handler with the database
	handler := handler.NewHandler(database)
	if err := handler.Router.RunTLS(":8082", certFile, keyFile); err != nil {
		log.Fatalf("Failed to start the server: %v", err)
	}
}
