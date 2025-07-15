package routes

import (
	"os"

	"github.com/Ananth-NQI/truckpe-backend/internal/handlers"
	"github.com/Ananth-NQI/truckpe-backend/internal/middleware"
	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, store storage.Store, twilioService *services.TwilioService) {

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to TruckPe Backend!",
			"version": "1.0.0",
			"endpoints": fiber.Map{
				"health":        "/health",
				"api":           "/api",
				"webhook":       "/webhook/whatsapp",
				"test_whatsapp": "/test/whatsapp",
				"admin":         "/admin",
			},
		})
	})

	// Health check - You'll need to implement this or use a simple handler
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// API routes
	api := app.Group("/api")

	// Trucker routes - These need to be implemented in handlers package
	truckers := api.Group("/truckers")
	truckers.Post("/register", func(c *fiber.Ctx) error {
		// TODO: Implement in handlers
		return c.JSON(fiber.Map{"error": "not implemented"})
	})
	truckers.Get("/:id", func(c *fiber.Ctx) error {
		// TODO: Implement in handlers
		return c.JSON(fiber.Map{"error": "not implemented"})
	})

	// Load routes - These need to be implemented
	loads := api.Group("/loads")
	loads.Get("/", func(c *fiber.Ctx) error {
		// TODO: Implement in handlers
		return c.JSON(fiber.Map{"error": "not implemented"})
	})

	// Booking routes - These need to be implemented
	bookings := api.Group("/bookings")
	bookings.Post("/", func(c *fiber.Ctx) error {
		// TODO: Implement in handlers
		return c.JSON(fiber.Map{"error": "not implemented"})
	})

	// ========== WEBHOOK ROUTES ==========
	webhooks := app.Group("/webhook")

	// WhatsApp webhook - ENVIRONMENT-AWARE VALIDATION
	if os.Getenv("ENVIRONMENT") == "development" || os.Getenv("DISABLE_WEBHOOK_VALIDATION") == "true" {
		// Development: Skip validation for ngrok
		webhooks.Post("/whatsapp", handlers.HandleWebhook)
		// Log that validation is disabled
		if os.Getenv("ENVIRONMENT") == "development" {
			println("⚠️  WhatsApp webhook validation DISABLED for development")
		}
	} else {
		// Production: Validate webhook signature
		webhooks.Post("/whatsapp", middleware.ValidateTwilioSignature(), handlers.HandleWebhook)
	}

	// ========== TEST ROUTES (Development Only) ==========
	// Test WhatsApp endpoint (for development)
	app.Post("/test/whatsapp", handlers.TestWebhook)

	// ========== ADMIN ROUTES ==========
	admin := app.Group("/admin")
	admin.Get("/overview", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Admin routes not implemented yet"})
	})
}
