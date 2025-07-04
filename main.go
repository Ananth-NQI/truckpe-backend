package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/Ananth-NQI/truckpe-backend/database"
	"github.com/Ananth-NQI/truckpe-backend/internal/models" // Fixed: added 'internal'
	"github.com/Ananth-NQI/truckpe-backend/internal/routes"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

func main() {
	// Load .env file for local development
	if os.Getenv("INSTANCE_CONNECTION_NAME") == "" {
		err := godotenv.Load()
		if err != nil {
			log.Println("No .env file found - assuming production environment")
		}
	}

	// Initialize storage
	var store storage.Store

	// Check if we should use memory store (for testing)
	if os.Getenv("USE_MEMORY_STORE") == "true" {
		log.Println("⚠️  Using in-memory storage (not for production!)")
		store = storage.NewMemoryStore()
	} else {
		// Connect to database
		log.Println("📦 Connecting to PostgreSQL database...")
		database.Connect()

		// Run migrations
		log.Println("🔄 Running database migrations...")
		err := database.DB.AutoMigrate(
			&models.Trucker{},
			&models.Load{},
			&models.Booking{},
			&models.WhatsAppSession{},
		)
		if err != nil {
			log.Fatal("Failed to migrate database:", err)
		}
		log.Println("✅ Database migrations completed!")

		// Use database store
		store = storage.NewDatabaseStore(database.DB)
		log.Println("✅ Using PostgreSQL database storage")
	}

	// Create fiber app
	app := fiber.New(fiber.Config{
		AppName: "TruckPe Backend v1.0.0",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Health check endpoint with database status
	app.Get("/", func(c *fiber.Ctx) error {
		response := fiber.Map{
			"service":     "TruckPe Backend API",
			"version":     "1.0.0",
			"status":      "healthy",
			"environment": getEnvironment(),
			"storage":     getStorageType(),
		}

		// Add database status if using database
		if os.Getenv("USE_MEMORY_STORE") != "true" && database.DB != nil {
			sqlDB, err := database.DB.DB()
			dbStatus := "connected"
			if err != nil {
				dbStatus = "error: " + err.Error()
			} else if err := sqlDB.Ping(); err != nil {
				dbStatus = "error: " + err.Error()
			}

			// Get counts
			var truckerCount, loadCount, bookingCount int64
			database.DB.Model(&models.Trucker{}).Count(&truckerCount)
			database.DB.Model(&models.Load{}).Count(&loadCount)
			database.DB.Model(&models.Booking{}).Count(&bookingCount)

			response["database"] = fiber.Map{
				"status":   dbStatus,
				"truckers": truckerCount,
				"loads":    loadCount,
				"bookings": bookingCount,
			}
		}

		return c.JSON(response)
	})

	// Health check endpoint for monitoring
	app.Get("/health", func(c *fiber.Ctx) error {
		status := "healthy"
		statusCode := 200

		// Check database if using it
		if os.Getenv("USE_MEMORY_STORE") != "true" && database.DB != nil {
			sqlDB, err := database.DB.DB()
			if err != nil || sqlDB.Ping() != nil {
				status = "unhealthy"
				statusCode = 503
			}
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"status": status,
		})
	})

	// Setup routes
	routes.SetupRoutes(app, store)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Println("========================================")
	log.Printf("🚀 TruckPe Backend starting on port %s", port)
	log.Printf("📊 Storage: %s", getStorageType())
	log.Printf("🌍 Environment: %s", getEnvironment())
	log.Println("========================================")
	log.Println("✅ TEST: Logging is working!")

	log.Fatal(app.Listen(":" + port))
}

func getEnvironment() string {
	if os.Getenv("INSTANCE_CONNECTION_NAME") != "" {
		return "Production (Cloud Run)"
	}
	return "Development (Local)"
}

func getStorageType() string {
	if os.Getenv("USE_MEMORY_STORE") == "true" {
		return "In-Memory (Testing)"
	}
	return "PostgreSQL Database"
}
