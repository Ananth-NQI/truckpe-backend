package models

import "time"

// Booking represents a confirmed match between trucker and load
type Booking struct {
	ID        string `json:"id"`
	LoadID    string `json:"load_id"`
	TruckerID string `json:"trucker_id"`
	ShipperID string `json:"shipper_id"`

	// Pricing
	AgreedPrice float64 `json:"agreed_price"`
	Commission  float64 `json:"commission"` // TruckPe's 5% commission
	NetAmount   float64 `json:"net_amount"` // Amount trucker receives

	// Status tracking
	Status string `json:"status"` // "confirmed", "trucker_assigned", "in_transit", "delivered", "completed"

	// Payment status
	PaymentStatus string `json:"payment_status"` // "pending", "escrow", "released", "completed"
	PaymentID     string `json:"payment_id"`     // Razorpay payment ID

	// Tracking
	OTP    string `json:"-"`       // Hidden in JSON, for delivery confirmation
	PodURL string `json:"pod_url"` // Proof of Delivery document

	// Timestamps
	ConfirmedAt *time.Time `json:"confirmed_at"`
	PickedUpAt  *time.Time `json:"picked_up_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	CompletedAt *time.Time `json:"completed_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BookingStatus constants
const (
	BookingStatusConfirmed       = "confirmed"
	BookingStatusTruckerAssigned = "trucker_assigned"
	BookingStatusInTransit       = "in_transit"
	BookingStatusDelivered       = "delivered"
	BookingStatusCompleted       = "completed"

	PaymentStatusPending   = "pending"
	PaymentStatusEscrow    = "escrow"
	PaymentStatusReleased  = "released"
	PaymentStatusCompleted = "completed"
)
