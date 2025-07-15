package handlers

import (
	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

type PaymentHandler struct {
	store         storage.Store
	twilioService *services.TwilioService
}

func NewPaymentHandler(store storage.Store, twilioService *services.TwilioService) *PaymentHandler {
	return &PaymentHandler{
		store:         store,
		twilioService: twilioService,
	}
}

func (h *PaymentHandler) GetPaymentSummary(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Payment summary endpoint - not implemented yet",
	})
}

func (h *PaymentHandler) ProcessPayment(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Process payment endpoint - not implemented yet",
	})
}

func (h *PaymentHandler) GetPendingPayments(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Pending payments endpoint - not implemented yet",
	})
}

func (h *PaymentHandler) HandleWebhook(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Payment webhook endpoint - not implemented yet",
	})
}

func (h *PaymentHandler) HandleTestWebhook(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Test payment webhook endpoint - not implemented yet",
	})
}
