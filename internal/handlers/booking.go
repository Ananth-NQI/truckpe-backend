package handlers

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// BookingHandler handles booking-related requests
type BookingHandler struct {
	store *storage.MemoryStore
}

// NewBookingHandler creates a new booking handler
func NewBookingHandler(store *storage.MemoryStore) *BookingHandler {
	return &BookingHandler{
		store: store,
	}
}

// CreateBooking handles creating a new booking
func (h *BookingHandler) CreateBooking(c *fiber.Ctx) error {
	var req struct {
		LoadID    string `json:"load_id"`
		TruckerID string `json:"trucker_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.LoadID == "" || req.TruckerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Load ID and Trucker ID are required",
		})
	}

	// Create booking
	booking, err := h.store.CreateBooking(req.LoadID, req.TruckerID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Booking created successfully",
		"booking": booking,
	})
}

// GetBooking retrieves booking by ID
func (h *BookingHandler) GetBooking(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Booking ID is required",
		})
	}

	booking, err := h.store.GetBooking(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Booking not found",
		})
	}

	return c.JSON(booking)
}
