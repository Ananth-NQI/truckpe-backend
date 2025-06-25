package handlers

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// TruckerHandler handles trucker-related requests
type TruckerHandler struct {
	store *storage.MemoryStore
}

// NewTruckerHandler creates a new trucker handler
func NewTruckerHandler(store *storage.MemoryStore) *TruckerHandler {
	return &TruckerHandler{
		store: store,
	}
}

// Register handles trucker registration
func (h *TruckerHandler) Register(c *fiber.Ctx) error {
	var reg models.TruckerRegistration

	if err := c.BodyParser(&reg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Basic validation
	if reg.Name == "" || reg.Phone == "" || reg.VehicleNo == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name, phone, and vehicle number are required",
		})
	}

	// Create trucker
	trucker, err := h.store.CreateTrucker(&reg)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register trucker",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Trucker registered successfully",
		"trucker": trucker,
	})
}

// GetTrucker retrieves trucker by ID
func (h *TruckerHandler) GetTrucker(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Trucker ID is required",
		})
	}

	trucker, err := h.store.GetTrucker(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Trucker not found",
		})
	}

	return c.JSON(trucker)
}
