package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/Ananth-NQI/truckpe-backend/database"
	"github.com/Ananth-NQI/truckpe-backend/internal/jobs"
	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/routes"
	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

func main() {
	// Load .env file for local development
	if os.Getenv("INSTANCE_CONNECTION_NAME") == "" {
		// Try multiple locations for .env file
		err := godotenv.Load(".env")
		if err != nil {
			err = godotenv.Load("environments/.env.development")
			if err != nil {
				log.Println("⚠️  No .env file found - checking environment variables")
			}
		}

		// Debug what we loaded
		log.Printf("🔍 TWILIO_ACCOUNT_SID exists: %v", os.Getenv("TWILIO_ACCOUNT_SID") != "")
		log.Printf("🔍 TWILIO_AUTH_TOKEN exists: %v", os.Getenv("TWILIO_AUTH_TOKEN") != "")
		log.Printf("🔍 TWILIO_WHATSAPP_FROM: %s", os.Getenv("TWILIO_WHATSAPP_FROM"))
	}

	// Get Twilio credentials
	twilioAccountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken := os.Getenv("TWILIO_AUTH_TOKEN")
	twilioPhoneNumber := os.Getenv("TWILIO_PHONE_NUMBER")

	if twilioAccountSID == "" || twilioAuthToken == "" || twilioPhoneNumber == "" {
		log.Println("⚠️  Twilio credentials not found - WhatsApp features will be limited")
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
			&models.Shipper{},
			&models.OTP{},
			&models.SupportTicket{}, // Add new models
			&models.Verification{},  // Add new models
			&models.TruckerStats{},  // Add new models
			&models.ShipperStats{},  // Add new models
		)
		if err != nil {
			log.Fatal("Failed to migrate database:", err)
		}
		log.Println("✅ Database migrations completed!")

		// Use database store
		store = storage.NewDatabaseStore(database.DB)
		log.Println("✅ Using PostgreSQL database storage")
	}

	// Initialize Twilio service
	twilioService, err := services.NewTwilioService()
	if err != nil {
		log.Fatal("Failed to initialize Twilio service:", err)
	}
	log.Println("✅ Twilio service initialized")

	// Set global instances
	storage.SetStore(store)
	services.SetTwilioService(twilioService)

	// Initialize all services
	paymentService := services.NewPaymentService(store, twilioService)
	sessionManager := services.NewSessionManager(store, twilioService)
	services.SetSessionManager(sessionManager)
	routeSuggestionService := services.NewRouteSuggestionService(store, twilioService)
	interactiveService := services.NewInteractiveTemplateService(store, twilioService)
	_ = interactiveService // Mark as intentionally unused for now

	// Initialize and start notification jobs
	notificationJob := jobs.NewNotificationJob(store, twilioService)
	notificationJob.Start()

	// Start scheduled services
	paymentService.SchedulePaymentReminders()
	routeSuggestionService.ScheduleRouteSuggestions()

	log.Println("✅ All services initialized and scheduled jobs started")

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
			"whatsapp": fiber.Map{
				"configured": twilioAccountSID != "",
				"templates":  41,
			},
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
			var truckerCount, loadCount, bookingCount, shipperCount, otpCount int64
			database.DB.Model(&models.Trucker{}).Count(&truckerCount)
			database.DB.Model(&models.Load{}).Count(&loadCount)
			database.DB.Model(&models.Booking{}).Count(&bookingCount)
			database.DB.Model(&models.Shipper{}).Count(&shipperCount)
			database.DB.Model(&models.OTP{}).Count(&otpCount)

			response["database"] = fiber.Map{
				"status":   dbStatus,
				"truckers": truckerCount,
				"loads":    loadCount,
				"bookings": bookingCount,
				"shippers": shipperCount,
				"otps":     otpCount,
			}
		}

		// Add service status
		response["services"] = fiber.Map{
			"payment":       "active",
			"sessions":      len(sessionManager.GetActiveSessions()),
			"notifications": "running",
			"scheduled_jobs": fiber.Map{
				"payment_reminders":  "active",
				"route_suggestions":  "active",
				"weekly_summaries":   "active",
				"document_expiry":    "active",
				"maintenance_alerts": "active",
				"inactivity_check":   "active",
			},
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

		// Check Twilio service
		twilioHealthy := twilioService != nil && twilioAccountSID != ""

		return c.Status(statusCode).JSON(fiber.Map{
			"status": status,
			"services": fiber.Map{
				"database": status == "healthy",
				"twilio":   twilioHealthy,
			},
		})
	})

	// Setup routes with twilioService
	routes.SetupRoutes(app, store, twilioService)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("\n🛑 Gracefully shutting down...")
		log.Println("⏹️  Stopping notification jobs...")
		notificationJob.Stop()
		log.Println("⏹️  Shutting down server...")
		_ = app.Shutdown()
	}()

	// Start server
	log.Println("========================================")
	log.Printf("🚀 TruckPe Backend starting on port %s", port)
	log.Printf("📊 Storage: %s", getStorageType())
	log.Printf("🌍 Environment: %s", getEnvironment())
	log.Printf("📱 WhatsApp: %s", getWhatsAppStatus(twilioAccountSID))
	log.Printf("📋 Templates: 41 integrated")
	log.Println("========================================")
	log.Println("✅ TEST: Logging is working!")

	// Log active services
	log.Println("🔧 Active Services:")
	log.Println("  ✓ Payment processing & reminders")
	log.Println("  ✓ Session management")
	log.Println("  ✓ Route suggestions")
	log.Println("  ✓ Interactive templates")
	log.Println("  ✓ Scheduled notifications")
	log.Println("========================================")

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

func getWhatsAppStatus(twilioSID string) string {
	if twilioSID == "" {
		return "Not configured"
	}
	return "Configured"
}
