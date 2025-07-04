package handlers

import (
	"log"

	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// WhatsAppHandler handles WhatsApp webhook requests
// WhatsAppHandler handles WhatsApp webhook requests
type WhatsAppHandler struct {
	store           storage.Store
	whatsappService *services.WhatsAppService
	twilioService   *services.TwilioService // ADD THIS
}

// NewWhatsAppHandler creates a new WhatsApp handler
func NewWhatsAppHandler(store storage.Store) *WhatsAppHandler {
	// Initialize Twilio service
	twilioSvc, err := services.NewTwilioService()
	if err != nil {
		log.Printf("⚠️  Warning: Twilio service not initialized: %v", err)
		// Continue without Twilio for testing
	}

	return &WhatsAppHandler{
		store:           store,
		whatsappService: services.NewWhatsAppService(store),
		twilioService:   twilioSvc,
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
		// Remove 'whatsapp:' prefix if present
		from := payload.From
		if len(from) > 9 && from[:9] == "whatsapp:" {
			from = from[9:]
		}

		// Process the message
		response, err := h.whatsappService.ProcessMessage(from, payload.Body)
		if err != nil {
			log.Printf("Error processing message: %v", err)
			response = "❌ Sorry, something went wrong. Please try again."
		}

		// Send the response back via Twilio
		if h.twilioService != nil && response != "" {
			err = h.twilioService.SendWhatsAppMessage(from, response)
			if err != nil {
				log.Printf("❌ Failed to send WhatsApp response: %v", err)
			} else {
				log.Printf("✅ Response sent to %s", from)
			}
		} else {
			log.Printf("📤 Response (not sent - Twilio not configured): %s", response)
		}
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

	// ADD THESE LOGS
	log.Printf("🧪 Test webhook received from %s: %s", payload.From, payload.Message)

	// Process the message
	response, err := h.whatsappService.ProcessMessage(payload.From, payload.Message)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		response = "❌ Sorry, something went wrong. Please try again."
	}

	// ADD THIS LOG
	log.Printf("📤 Test response generated: %s", response)

	// ADD THIS CHECK
	if h.twilioService != nil {
		log.Println("✅ Twilio service is initialized")
	} else {
		log.Println("⚠️ Twilio service is NOT initialized")
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"response": response,
	})
}
