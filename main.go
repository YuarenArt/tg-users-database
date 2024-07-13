package main

import (
	"log"
	"tg-users-database/pkg/db"
	"tg-users-database/pkg/handler"
)

// TODO use RESTful
// TODO generate documentation with swagger
// TODO создать обработчик для документирования

// @title User Database API
// @version 1.0
// @description This is a sample server for managing user subscriptions.

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
