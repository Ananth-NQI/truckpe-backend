package handlers

import (
	"log"

	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// WhatsAppHandler handles WhatsApp webhook requests
type WhatsAppHandler struct {
	whatsappService *services.WhatsAppService
}

// NewWhatsAppHandler creates a new WhatsApp handler
func NewWhatsAppHandler(whatsappService *services.WhatsAppService) *WhatsAppHandler {
	return &WhatsAppHandler{
		whatsappService: whatsappService,
	}
}

// HandleWebhook processes incoming WhatsApp messages
func (h *WhatsAppHandler) HandleWebhook(c *fiber.Ctx) error {
	// Twilio sends different payloads for different events
	var payload TwilioWebhookPayload

	if err := c.BodyParser(&payload); err != nil {
		log.Printf("Error parsing webhook: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook payload",
		})
	}

	// Log incoming message
	log.Printf("📱 WhatsApp Message from %s: %s", payload.From, payload.Body)

	// Process only incoming messages (not status updates)
	if payload.Body != "" && payload.From != "" {
		// Process the message
		response, err := h.whatsappService.ProcessMessage(payload.From, payload.Body)
		if err != nil {
			log.Printf("Error processing message: %v", err)
			response = "❌ Sorry, something went wrong. Please try again."
		}

		// Log the response (in production, you'd send this back via Twilio API)
		log.Printf("📤 Response: %s", response)

		// For now, we'll return the response in the webhook response
		// In production, you'd use Twilio API to send the message
		return c.JSON(fiber.Map{
			"message":  "Message processed",
			"response": response,
		})
	}

	// Acknowledge webhook receipt
	return c.SendStatus(fiber.StatusOK)
}

// TwilioWebhookPayload represents incoming WhatsApp message from Twilio
type TwilioWebhookPayload struct {
	MessageSid          string `form:"MessageSid"`
	AccountSid          string `form:"AccountSid"`
	MessagingServiceSid string `form:"MessagingServiceSid"`
	From                string `form:"From"` // WhatsApp number (whatsapp:+919876543210)
	To                  string `form:"To"`   // Your Twilio number
	Body                string `form:"Body"` // Message text
	NumMedia            string `form:"NumMedia"`
	MediaUrl0           string `form:"MediaUrl0"`
	MediaContentType0   string `form:"MediaContentType0"`
}

// For testing without Twilio
type TestWebhookPayload struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

// HandleTestWebhook processes test WhatsApp messages (for development)
func (h *WhatsAppHandler) HandleTestWebhook(c *fiber.Ctx) error {
	var payload TestWebhookPayload

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid test payload",
		})
	}

	// Process the message
	response, err := h.whatsappService.ProcessMessage(payload.From, payload.Message)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		response = "❌ Sorry, something went wrong. Please try again."
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"response": response,
	})
}
