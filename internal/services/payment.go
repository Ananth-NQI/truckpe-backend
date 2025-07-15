package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// PaymentService handles payment processing and notifications
type PaymentService struct {
	store         storage.Store
	twilioService *TwilioService
}

// NewPaymentService creates a new payment service
func NewPaymentService(store storage.Store, twilioService *TwilioService) *PaymentService {
	return &PaymentService{
		store:         store,
		twilioService: twilioService,
	}
}

// RazorpayWebhookPayload represents the webhook data from Razorpay
type RazorpayWebhookPayload struct {
	Event     string                 `json:"event"`
	Entity    string                 `json:"entity"`
	Contains  []string               `json:"contains"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt int64                  `json:"created_at"`
}

// ProcessPaymentWebhook handles payment gateway webhooks
func (p *PaymentService) ProcessPaymentWebhook(payload []byte) error {
	var webhook RazorpayWebhookPayload
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return fmt.Errorf("failed to parse webhook: %v", err)
	}

	log.Printf("Processing payment webhook: %s", webhook.Event)

	switch webhook.Event {
	case "payment.captured":
		return p.handlePaymentCaptured(webhook.Payload)
	case "payment.failed":
		return p.handlePaymentFailed(webhook.Payload)
	default:
		log.Printf("Unhandled webhook event: %s", webhook.Event)
		return nil
	}
}

// handlePaymentCaptured processes successful payments
func (p *PaymentService) handlePaymentCaptured(payload map[string]interface{}) error {
	payment := payload["payment"].(map[string]interface{})

	// Extract payment details
	paymentID := payment["id"].(string)
	amount := payment["amount"].(float64) / 100 // Convert paise to rupees

	// Get notes which should contain booking_id
	notes := payment["notes"].(map[string]interface{})
	bookingID, ok := notes["booking_id"].(string)
	if !ok {
		return fmt.Errorf("booking_id not found in payment notes")
	}

	// Get booking
	booking, err := p.store.GetBooking(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %v", err)
	}

	// Update payment status
	booking.PaymentStatus = "completed"
	booking.PaymentID = paymentID
	booking.PaidAt = &[]time.Time{time.Now()}[0]

	// Update booking
	if err := p.store.UpdateBooking(booking); err != nil {
		return fmt.Errorf("failed to update booking: %v", err)
	}

	// Get trucker details
	trucker, err := p.store.GetTruckerByID(booking.TruckerID)
	if err != nil {
		return fmt.Errorf("trucker not found: %v", err)
	}

	// Send payment confirmation to trucker
	templateService := NewTemplateService(p.twilioService)
	params := map[string]string{
		"amount":       fmt.Sprintf("₹%.0f", amount),
		"payment_id":   paymentID,
		"booking_id":   bookingID,
		"trucker_name": trucker.Name,
	}

	err = templateService.SendTemplate(trucker.Phone, "payment_processed", params)
	if err != nil {
		log.Printf("Failed to send payment template to trucker: %v", err)
		// Don't fail the webhook processing
	}

	// Get load details for shipper notification
	load, _ := p.store.GetLoad(booking.LoadID)
	if load != nil && load.ShipperPhone != "" {
		shipperParams := map[string]string{
			"amount":       fmt.Sprintf("₹%.0f", booking.AgreedPrice),
			"booking_id":   bookingID,
			"trucker_name": trucker.Name,
		}
		_ = templateService.SendTemplate(load.ShipperPhone, "payment_processed", shipperParams)
	}

	log.Printf("Payment processed successfully: %s for booking %s", paymentID, bookingID)
	return nil
}

// handlePaymentFailed processes failed payments
func (p *PaymentService) handlePaymentFailed(payload map[string]interface{}) error {
	payment := payload["payment"].(map[string]interface{})

	// Extract payment details
	paymentID := payment["id"].(string)
	errorCode := payment["error_code"].(string)
	errorDesc := payment["error_description"].(string)

	// Get notes which should contain booking_id
	notes := payment["notes"].(map[string]interface{})
	bookingID, ok := notes["booking_id"].(string)
	if !ok {
		return fmt.Errorf("booking_id not found in payment notes")
	}

	// Log the failure
	log.Printf("Payment failed: %s for booking %s - %s: %s",
		paymentID, bookingID, errorCode, errorDesc)

	// You might want to update booking status or retry payment
	// For now, just log it
	return nil
}

// ProcessPaymentForBooking initiates payment for a completed booking
func (p *PaymentService) ProcessPaymentForBooking(bookingID string) error {
	// Get booking
	booking, err := p.store.GetBooking(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %v", err)
	}

	// Check if already paid
	if booking.PaymentStatus == "completed" {
		return fmt.Errorf("payment already completed for booking %s", bookingID)
	}

	// Check if delivered
	if booking.Status != models.BookingStatusDelivered {
		return fmt.Errorf("booking %s not yet delivered", bookingID)
	}

	// In production, you would:
	// 1. Create payment order with Razorpay
	// 2. Process the payment
	// 3. Wait for webhook confirmation

	// For now, simulate payment processing
	log.Printf("Initiating payment for booking %s, amount: ₹%.0f",
		bookingID, booking.NetAmount)

	// Update payment status to processing
	booking.PaymentStatus = "processing"
	return p.store.UpdateBooking(booking)
}

// SendPaymentReminders sends reminders for pending payments
func (p *PaymentService) SendPaymentReminders() error {
	// Get all bookings with pending payments
	bookings, err := p.store.GetBookingsByPaymentStatus("pending")
	if err != nil {
		return fmt.Errorf("failed to get pending payments: %v", err)
	}

	templateService := NewTemplateService(p.twilioService)
	remindersSent := 0

	for _, booking := range bookings {
		// Only remind for delivered bookings
		if booking.Status != models.BookingStatusDelivered {
			continue
		}

		// Check if delivered more than 48 hours ago
		if booking.DeliveredAt != nil {
			hoursSinceDelivery := time.Since(*booking.DeliveredAt).Hours()
			if hoursSinceDelivery < 48 {
				continue // Payment window still open
			}
		}

		// Get trucker details
		trucker, err := p.store.GetTruckerByID(booking.TruckerID)
		if err != nil {
			log.Printf("Failed to get trucker %s: %v", booking.TruckerID, err)
			continue
		}

		// Send payment reminder
		params := map[string]string{
			"amount":       fmt.Sprintf("₹%.0f", booking.NetAmount),
			"booking_id":   booking.BookingID,
			"days_pending": fmt.Sprintf("%.0f", time.Since(*booking.DeliveredAt).Hours()/24),
		}

		err = templateService.SendTemplate(trucker.Phone, "payment_reminder", params)
		if err != nil {
			log.Printf("Failed to send payment reminder to %s: %v", trucker.Phone, err)
			continue
		}

		remindersSent++
		log.Printf("Payment reminder sent for booking %s", booking.BookingID)

		// Update last reminder sent time (you'd need to add this field to your model)
		// booking.LastPaymentReminderAt = &[]time.Time{time.Now()}[0]
		// p.store.UpdateBooking(booking)
	}

	log.Printf("Sent %d payment reminders", remindersSent)
	return nil
}

// GetPaymentSummary returns payment summary for a user
func (p *PaymentService) GetPaymentSummary(userPhone string) (*PaymentSummary, error) {
	// Check if trucker or shipper
	trucker, _ := p.store.GetTruckerByPhone(userPhone)
	if trucker != nil {
		return p.getTruckerPaymentSummary(trucker.TruckerID)
	}

	shipper, _ := p.store.GetShipperByPhone(userPhone)
	if shipper != nil {
		return p.getShipperPaymentSummary(shipper.ShipperID)
	}

	return nil, fmt.Errorf("user not found")
}

// PaymentSummary represents payment statistics
type PaymentSummary struct {
	TotalEarned    float64
	TotalPending   float64
	TotalPaid      float64
	LastPayment    *time.Time
	PendingCount   int
	CompletedCount int
}

// getTruckerPaymentSummary gets payment summary for a trucker
func (p *PaymentService) getTruckerPaymentSummary(truckerID string) (*PaymentSummary, error) {
	bookings, err := p.store.GetBookingsByTrucker(truckerID)
	if err != nil {
		return nil, err
	}

	summary := &PaymentSummary{}

	for _, booking := range bookings {
		if booking.Status == models.BookingStatusDelivered {
			summary.TotalEarned += booking.NetAmount

			if booking.PaymentStatus == "completed" {
				summary.TotalPaid += booking.NetAmount
				summary.CompletedCount++
				if booking.PaidAt != nil && (summary.LastPayment == nil || booking.PaidAt.After(*summary.LastPayment)) {
					summary.LastPayment = booking.PaidAt
				}
			} else if booking.PaymentStatus == "pending" {
				summary.TotalPending += booking.NetAmount
				summary.PendingCount++
			}
		}
	}

	return summary, nil
}

// getShipperPaymentSummary gets payment summary for a shipper
func (p *PaymentService) getShipperPaymentSummary(shipperID string) (*PaymentSummary, error) {
	loads, err := p.store.GetLoadsByShipper(shipperID)
	if err != nil {
		return nil, err
	}

	summary := &PaymentSummary{}

	for _, load := range loads {
		bookings, _ := p.store.GetBookingsByLoad(load.LoadID)
		for _, booking := range bookings {
			if booking.Status == models.BookingStatusDelivered {
				summary.TotalEarned += booking.AgreedPrice

				if booking.PaymentStatus == "completed" {
					summary.TotalPaid += booking.AgreedPrice
					summary.CompletedCount++
				} else if booking.PaymentStatus == "pending" {
					summary.TotalPending += booking.AgreedPrice
					summary.PendingCount++
				}
			}
		}
	}

	return summary, nil
}

// SchedulePaymentReminders sets up scheduled payment reminders
func (p *PaymentService) SchedulePaymentReminders() {
	// Run every day at 10 AM
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			duration := next.Sub(now)
			log.Printf("Next payment reminder run in %v", duration)

			time.Sleep(duration)

			if err := p.SendPaymentReminders(); err != nil {
				log.Printf("Error sending payment reminders: %v", err)
			}
		}
	}()
}
