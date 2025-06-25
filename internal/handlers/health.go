package handlers

import "github.com/gofiber/fiber/v2"

// HealthHandler handles health check requests
type HealthHandler struct {
	Version string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{
		Version: version,
	}
}

// Check returns the health status of the service
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "OK",
		"service": "TruckPe Backend",
		"version": h.Version,
	})
}
