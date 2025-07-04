package handlers

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// BookingHandler handles booking-related requests
type BookingHandler struct {
	store storage.Store // Changed from *storage.MemoryStore to interface
}

// NewBookingHandler creates a new booking handler
func NewBookingHandler(store storage.Store) *BookingHandler { // Changed parameter type
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
		// Handle specific errors
		if err.Error() == "load not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Load not found",
			})
		}
		if err.Error() == "trucker not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Trucker not found",
			})
		}
		if err.Error() == "load not available" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Load is not available for booking",
			})
		}
		if err.Error() == "trucker not available" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Trucker is not available",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create booking",
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

// GetTruckerBookings retrieves all bookings for a trucker
func (h *BookingHandler) GetTruckerBookings(c *fiber.Ctx) error {
	truckerID := c.Params("truckerID")
	if truckerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Trucker ID is required",
		})
	}

	bookings, err := h.store.GetBookingsByTrucker(truckerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve bookings",
		})
	}

	return c.JSON(fiber.Map{
		"bookings": bookings,
		"count":    len(bookings),
	})
}

// GetLoadBookings retrieves all bookings for a load
func (h *BookingHandler) GetLoadBookings(c *fiber.Ctx) error {
	loadID := c.Params("loadID")
	if loadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Load ID is required",
		})
	}

	bookings, err := h.store.GetBookingsByLoad(loadID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve bookings",
		})
	}

	return c.JSON(fiber.Map{
		"bookings": bookings,
		"count":    len(bookings),
	})
}

// UpdateBookingStatus updates the status of a booking
func (h *BookingHandler) UpdateBookingStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Booking ID is required",
		})
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate status
	validStatuses := map[string]bool{
		models.BookingStatusConfirmed:       true,
		models.BookingStatusTruckerAssigned: true,
		models.BookingStatusInTransit:       true,
		models.BookingStatusDelivered:       true,
		models.BookingStatusCompleted:       true,
	}

	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid status value",
		})
	}

	if err := h.store.UpdateBookingStatus(id, req.Status); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Failed to update booking status",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Booking status updated successfully",
	})
}
