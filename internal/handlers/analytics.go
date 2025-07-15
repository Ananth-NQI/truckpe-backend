package handlers

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

type AnalyticsHandler struct {
	store storage.Store
}

func NewAnalyticsHandler(store storage.Store) *AnalyticsHandler {
	return &AnalyticsHandler{
		store: store,
	}
}

func (h *AnalyticsHandler) GetTruckerStats(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Trucker stats endpoint - not implemented yet",
	})
}

func (h *AnalyticsHandler) GetShipperStats(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Shipper stats endpoint - not implemented yet",
	})
}

func (h *AnalyticsHandler) GetWeeklySummary(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Weekly summary endpoint - not implemented yet",
	})
}
