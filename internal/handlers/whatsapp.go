package handlers

import (
	"log"
	"os"

	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// HandleWebhook processes incoming WhatsApp messages from Twilio
func HandleWebhook(c *fiber.Ctx) error {
	// Parse form values from Twilio
	from := c.FormValue("From")
	body := c.FormValue("Body")

	// Check for button/interactive responses
	buttonPayload := c.FormValue("ButtonPayload", "")
	listReplyId := c.FormValue("ListReplyId", "")

	// Combine button responses
	if buttonPayload == "" && listReplyId != "" {
		buttonPayload = listReplyId
	}

	// Log the webhook data
	log.Printf("WhatsApp webhook - From: %s, Body: %s, ButtonPayload: %s", from, body, buttonPayload)

	// Get services
	store := storage.GetStore()
	twilioService := services.GetTwilioService()

	// Check if natural flow is enabled (can be controlled via env var)
	useNaturalFlow := os.Getenv("USE_NATURAL_FLOW") != "false" // Default to true

	if useNaturalFlow {
		// Initialize all required services
		templateService := services.NewTemplateService(twilioService)
		interactiveService := services.NewInteractiveTemplateService(store, twilioService)
		sessionManager := services.GetSessionManager() // Use singleton

		// Create natural flow service
		naturalFlowService := services.NewNaturalFlowService(
			store,
			sessionManager,
			templateService,
			interactiveService,
			twilioService,
		)

		// Process through natural flow
		err := naturalFlowService.ProcessNaturalMessage(from, body, buttonPayload)
		if err != nil {
			log.Printf("Natural flow error: %v", err)
			// Fallback to command-based processing
			whatsappService := services.NewWhatsAppService(store, twilioService)
			response, _ := whatsappService.ProcessMessage(from, body)
			if response != "" {
				twilioService.SendWhatsAppMessage(from, response)
			}
		}
	} else {
		// Use existing command-based processing
		whatsappService := services.NewWhatsAppService(store, twilioService)
		response, err := whatsappService.ProcessMessage(from, body)
		if err != nil {
			log.Printf("Error processing message: %v", err)
			response = "Sorry, something went wrong. Please try again."
		}

		// Send response if any
		if response != "" {
			err = twilioService.SendWhatsAppMessage(from, response)
			if err != nil {
				log.Printf("Error sending response: %v", err)
			}
		}
	}

	// Return success to Twilio
	return c.SendStatus(fiber.StatusOK)
}

// TestWebhook is a test endpoint for local development
func TestWebhook(c *fiber.Ctx) error {
	// Parse JSON body for testing
	var testData struct {
		From          string `json:"from"`
		Body          string `json:"body"`
		ButtonPayload string `json:"button_payload"`
	}

	if err := c.BodyParser(&testData); err != nil {
		// Try form data
		testData.From = c.FormValue("from", "whatsapp:+1234567890")
		testData.Body = c.FormValue("body", "test message")
		testData.ButtonPayload = c.FormValue("button_payload", "")
	}

	// Get services
	store := storage.GetStore()
	twilioService := services.GetTwilioService()

	// Check if natural flow is enabled
	useNaturalFlow := os.Getenv("USE_NATURAL_FLOW") != "false"

	var response string
	var err error

	if useNaturalFlow {
		// Initialize all required services
		templateService := services.NewTemplateService(twilioService)
		interactiveService := services.NewInteractiveTemplateService(store, twilioService)
		sessionManager := services.GetSessionManager()

		// Create natural flow service
		naturalFlowService := services.NewNaturalFlowService(
			store,
			sessionManager,
			templateService,
			interactiveService,
			twilioService,
		)

		// Process through natural flow
		err = naturalFlowService.ProcessNaturalMessage(testData.From, testData.Body, testData.ButtonPayload)
		if err != nil {
			response = "Natural flow error: " + err.Error()
		} else {
			response = "Message processed through natural flow"
		}
	} else {
		// Use command-based processing
		whatsappService := services.NewWhatsAppService(store, twilioService)
		response, err = whatsappService.ProcessMessage(testData.From, testData.Body)
		if err != nil {
			response = "Error: " + err.Error()
		}
	}

	// Return response
	return c.JSON(fiber.Map{
		"success":      err == nil,
		"response":     response,
		"from":         testData.From,
		"body":         testData.Body,
		"button":       testData.ButtonPayload,
		"natural_flow": useNaturalFlow,
	})
}
