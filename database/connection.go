package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	var err error

	// Get environment variables
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPass := os.Getenv("DB_PASS")
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "truckpe"
	}

	// For Cloud Run with Cloud SQL
	socketDir := "/cloudsql"
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

	var dsn string
	if instanceConnectionName != "" {
		// Production: Connect via Unix socket
		dsn = fmt.Sprintf("host=%s/%s user=%s password=%s dbname=%s sslmode=disable",
			socketDir, instanceConnectionName, dbUser, dbPass, dbName)
		log.Printf("Connecting to Cloud SQL via socket: %s", instanceConnectionName)
	} else {
		// Local development: Connect via TCP
		dsn = fmt.Sprintf("host=localhost user=%s password=%s dbname=%s port=5432 sslmode=disable",
			dbUser, dbPass, dbName)
		log.Println("Connecting to local PostgreSQL")
	}

	// Connect to database
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		panic(err)
	}

	log.Println("✅ Database connected successfully!")
}
