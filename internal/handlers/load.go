package handlers

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// LoadHandler handles load-related requests
type LoadHandler struct {
	store storage.Store // Changed from *storage.MemoryStore to interface
}

// NewLoadHandler creates a new load handler
func NewLoadHandler(store storage.Store) *LoadHandler { // Changed parameter type
	return &LoadHandler{
		store: store,
	}
}

// CreateLoad handles creating a new load
func (h *LoadHandler) CreateLoad(c *fiber.Ctx) error {
	var load models.Load

	if err := c.BodyParser(&load); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Basic validation
	if load.FromCity == "" || load.ToCity == "" || load.Material == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "From city, to city, and material are required",
		})
	}

	if load.ShipperID == "" || load.ShipperName == "" || load.ShipperPhone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Shipper details are required",
		})
	}

	if load.Weight <= 0 || load.Price <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Weight and price must be greater than zero",
		})
	}

	// Create load
	createdLoad, err := h.store.CreateLoad(&load)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create load",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Load created successfully",
		"load":    createdLoad,
	})
}

// GetLoad retrieves a single load by ID
func (h *LoadHandler) GetLoad(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Load ID is required",
		})
	}

	load, err := h.store.GetLoad(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Load not found",
		})
	}

	return c.JSON(load)
}

// GetLoads retrieves all available loads
func (h *LoadHandler) GetLoads(c *fiber.Ctx) error {
	loads, err := h.store.GetAvailableLoads()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve loads",
		})
	}

	return c.JSON(fiber.Map{
		"loads": loads,
		"count": len(loads),
	})
}

// SearchLoads searches for loads based on criteria
func (h *LoadHandler) SearchLoads(c *fiber.Ctx) error {
	var search models.LoadSearch

	if err := c.BodyParser(&search); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid search parameters",
		})
	}

	results, err := h.store.SearchLoads(&search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search loads",
		})
	}

	return c.JSON(fiber.Map{
		"results": results,
		"count":   len(results),
	})
}

// UpdateLoadStatus updates the status of a load
func (h *LoadHandler) UpdateLoadStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Load ID is required",
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
		models.LoadStatusAvailable: true,
		models.LoadStatusBooked:    true,
		models.LoadStatusInTransit: true,
		models.LoadStatusDelivered: true,
	}

	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid status value",
		})
	}

	if err := h.store.UpdateLoadStatus(id, req.Status); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Failed to update load status",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Load status updated successfully",
	})
}
