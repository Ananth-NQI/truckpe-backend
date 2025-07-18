package routes

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/handlers"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, store storage.Store) { // Changed from *storage.MemoryStore to interface

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler("1.0.0")
	truckerHandler := handlers.NewTruckerHandler(store)
	loadHandler := handlers.NewLoadHandler(store)
	bookingHandler := handlers.NewBookingHandler(store)
	whatsappHandler := handlers.NewWhatsAppHandler(store)

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
	truckers.Get("/", truckerHandler.GetTruckerByPhone) // Query param: ?phone=+919876543210

	// Load routes
	loads := api.Group("/loads")
	loads.Get("/", loadHandler.GetLoads)
	loads.Post("/", loadHandler.CreateLoad)
	loads.Get("/:id", loadHandler.GetLoad)
	loads.Post("/search", loadHandler.SearchLoads)
	loads.Put("/:id/status", loadHandler.UpdateLoadStatus)

	// Booking routes
	bookings := api.Group("/bookings")
	bookings.Post("/", bookingHandler.CreateBooking)
	bookings.Get("/:id", bookingHandler.GetBooking)
	bookings.Get("/trucker/:truckerID", bookingHandler.GetTruckerBookings)
	bookings.Get("/load/:loadID", bookingHandler.GetLoadBookings)
	bookings.Put("/:id/status", bookingHandler.UpdateBookingStatus)

	// WhatsApp webhook (for production Twilio)
	app.Post("/webhook/whatsapp", whatsappHandler.HandleWebhook)

	// Test WhatsApp endpoint (for development)
	app.Post("/test/whatsapp", whatsappHandler.HandleTestWebhook)
}
