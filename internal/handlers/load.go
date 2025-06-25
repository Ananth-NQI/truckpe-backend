package handlers

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// LoadHandler handles load-related requests
type LoadHandler struct {
	store *storage.MemoryStore
}

// NewLoadHandler creates a new load handler
func NewLoadHandler(store *storage.MemoryStore) *LoadHandler {
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
