package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioService struct {
	client       *twilio.RestClient
	from         string // Your Twilio WhatsApp number
	whatsappFrom string
}

// NewTwilioService creates a new Twilio service instance
func NewTwilioService() (*TwilioService, error) {
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	from := os.Getenv("TWILIO_WHATSAPP_FROM") // Format: "whatsapp:+14155238886"

	if accountSid == "" || authToken == "" || from == "" {
		return nil, fmt.Errorf("missing Twilio credentials in environment variables")
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	return &TwilioService{
		client:       client,
		from:         from,
		whatsappFrom: from, // Initialize both with same value
	}, nil
}

// SendWhatsAppMessage sends a WhatsApp message via Twilio
func (t *TwilioService) SendWhatsAppMessage(to string, message string) error {
	params := &twilioApi.CreateMessageParams{}
	params.SetFrom(t.from)
	params.SetTo(fmt.Sprintf("whatsapp:%s", to))
	params.SetBody(message)

	resp, err := t.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("❌ Failed to send WhatsApp message: %v", err)
		return err
	}

	log.Printf("✅ WhatsApp message sent! SID: %s", *resp.Sid)
	return nil
}

// SendWhatsAppTemplate sends a WhatsApp template message via Twilio
func (t *TwilioService) SendWhatsAppTemplate(to string, templateSID string, contentVariables map[string]string) error {
	params := &twilioApi.CreateMessageParams{}
	params.SetFrom(t.from)
	params.SetTo(fmt.Sprintf("whatsapp:%s", to))

	// Set the content SID for the template
	params.SetContentSid(templateSID)

	// Set content variables (template parameters)
	// SetContentVariables expects a JSON string
	if len(contentVariables) > 0 {
		variablesJSON, err := json.Marshal(contentVariables)
		if err != nil {
			log.Printf("❌ Failed to marshal content variables: %v", err)
			return err
		}
		// SetContentVariables expects a string
		params.SetContentVariables(string(variablesJSON))
	}

	// Send the message
	resp, err := t.client.Api.CreateMessage(params)
	if err != nil {
		log.Printf("❌ Failed to send WhatsApp template: %v", err)
		return err
	}

	log.Printf("✅ WhatsApp template sent! SID: %s, Template: %s", *resp.Sid, templateSID)
	return nil
}

// SendWhatsAppInteractiveTemplate sends an interactive WhatsApp template
func (t *TwilioService) SendWhatsAppInteractiveTemplate(to string, templateSID string, contentVariables map[string]string, persistentAction map[string]interface{}) error {
	if t.client == nil {
		return fmt.Errorf("twilio client not initialized")
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(fmt.Sprintf("whatsapp:%s", to))
	params.SetFrom(t.whatsappFrom)

	// Set content SID for the template
	params.SetContentSid(templateSID)

	// Set content variables if provided
	// SetContentVariables expects a JSON string
	if len(contentVariables) > 0 {
		variablesJSON, err := json.Marshal(contentVariables)
		if err != nil {
			return fmt.Errorf("failed to marshal content variables: %w", err)
		}
		params.SetContentVariables(string(variablesJSON))
	}

	// Set persistent action (buttons/lists) if provided
	// SetPersistentAction expects []string
	if persistentAction != nil {
		persistentActionJSON, err := json.Marshal(persistentAction)
		if err != nil {
			return fmt.Errorf("failed to marshal persistent action: %w", err)
		}
		// SetPersistentAction expects []string, so we wrap the JSON string in a slice
		params.SetPersistentAction([]string{string(persistentActionJSON)})
	}

	// Send the message
	resp, err := t.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send interactive template: %w", err)
	}

	if resp.ErrorCode != nil && *resp.ErrorCode != 0 {
		return fmt.Errorf("twilio error %d: %s", *resp.ErrorCode, *resp.ErrorMessage)
	}

	log.Printf("Interactive template sent successfully to %s, SID: %s", to, *resp.Sid)
	return nil
}

// SendWhatsApp is an alias for SendWhatsAppMessage
func (t *TwilioService) SendWhatsApp(to string, message string) error {
	return t.SendWhatsAppMessage(to, message)
}
