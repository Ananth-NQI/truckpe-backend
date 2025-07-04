package services

import (
	"fmt"
	"log"
	"os"

	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioService struct {
	client *twilio.RestClient
	from   string // Your Twilio WhatsApp number
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
		client: client,
		from:   from,
	}, nil
}

// SendWhatsAppMessage sends a WhatsApp message via Twilio
func (t *TwilioService) SendWhatsAppMessage(to string, message string) error {
	params := &api.CreateMessageParams{}
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
