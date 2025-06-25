package routes

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/handlers"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, store *storage.MemoryStore) {
	// Initialize handlers
	healthHandler := handlers.NewHealthHandler("1.0.0")
	truckerHandler := handlers.NewTruckerHandler(store)
	loadHandler := handlers.NewLoadHandler(store)
	bookingHandler := handlers.NewBookingHandler(store)

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to TruckPe Backend!",
			"version": "1.0.0",
			"endpoints": fiber.Map{
				"health":  "/health",
				"api":     "/api",
				"webhook": "/webhook/whatsapp",
			},
		})
	})

	// Health check
	app.Get("/health", healthHandler.Check)

	// API routes
	api := app.Group("/api")

	// Trucker routes
	truckers := api.Group("/truckers")
	truckers.Post("/register", truckerHandler.Register)
	truckers.Get("/:id", truckerHandler.GetTrucker)

	// Load routes
	loads := api.Group("/loads")
	loads.Get("/", loadHandler.GetLoads)
	loads.Post("/", loadHandler.CreateLoad)
	loads.Get("/:id", loadHandler.GetLoad)
	loads.Post("/search", loadHandler.SearchLoads)

	// Booking routes
	bookings := api.Group("/bookings")
	bookings.Post("/", bookingHandler.CreateBooking)
	bookings.Get("/:id", bookingHandler.GetBooking)

	// WhatsApp webhook (placeholder for now)
	app.Post("/webhook/whatsapp", func(c *fiber.Ctx) error {
		// Log the webhook data
		var webhookData map[string]interface{}
		if err := c.BodyParser(&webhookData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid webhook data",
			})
		}

		// For now, just acknowledge
		return c.JSON(fiber.Map{
			"status": "received",
		})
	})
}
