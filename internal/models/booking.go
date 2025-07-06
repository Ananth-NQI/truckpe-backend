package models

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// Booking represents a confirmed match between trucker and load
type Booking struct {
	// Using gorm.Model gives us ID (uint), CreatedAt, UpdatedAt, DeletedAt automatically
	gorm.Model

	// Keep your BookingID as string for backward compatibility
	BookingID string `json:"booking_id" gorm:"uniqueIndex"`
	LoadID    string `json:"load_id" gorm:"index"`
	TruckerID string `json:"trucker_id" gorm:"index"`
	ShipperID string `json:"shipper_id" gorm:"index"`

	// Pricing (keeping all your fields)
	AgreedPrice float64 `json:"agreed_price"`
	Commission  float64 `json:"commission"` // TruckPe's 5% commission
	NetAmount   float64 `json:"net_amount"` // Amount trucker receives

	// Status tracking
	Status string `json:"status" gorm:"default:confirmed"` // "confirmed", "trucker_assigned", "in_transit", "delivered", "completed"

	// Payment status
	PaymentStatus string `json:"payment_status" gorm:"default:pending"` // "pending", "escrow", "released", "completed"
	PaymentID     string `json:"payment_id"`                            // Razorpay payment ID

	// Tracking
	// OTP removed - now handled by separate OTP table for better security
	PodURL string `json:"pod_url"` // Proof of Delivery document

	// Timestamps (keeping your custom timestamps)
	ConfirmedAt *time.Time `json:"confirmed_at"`
	PickedUpAt  *time.Time `json:"picked_up_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	CompletedAt *time.Time `json:"completed_at"`

	// Note: CreatedAt and UpdatedAt are automatically handled by gorm.Model

	// If you want to add relationships later (optional)
	// Load    Load    `json:"load,omitempty" gorm:"foreignKey:LoadID;references:LoadID"`
	// Trucker Trucker `json:"trucker,omitempty" gorm:"foreignKey:TruckerID;references:TruckerID"`
}

// BeforeCreate hook to auto-generate BookingID
func (b *Booking) BeforeCreate(tx *gorm.DB) error {
	// Generate BookingID if not set
	if b.BookingID == "" {
		b.BookingID = fmt.Sprintf("BK%d%03d", time.Now().Unix(), rand.Intn(1000))
	}

	// OTP generation removed - will be handled by OTP service when trucker arrives

	// Calculate net amount if not set
	if b.NetAmount == 0 && b.AgreedPrice > 0 {
		b.Commission = b.AgreedPrice * 0.05 // 5% commission
		b.NetAmount = b.AgreedPrice - b.Commission
	}

	// Set ConfirmedAt if not set
	if b.ConfirmedAt == nil {
		now := time.Now()
		b.ConfirmedAt = &now
	}

	return nil
}

// BookingStatus constants (KEEPING ALL YOUR CONSTANTS)
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

// Helper methods you can add
func (b *Booking) MarkAsPickedUp() {
	now := time.Now()
	b.PickedUpAt = &now
	b.Status = BookingStatusInTransit
}

func (b *Booking) MarkAsDelivered() {
	now := time.Now()
	b.DeliveredAt = &now
	b.Status = BookingStatusDelivered
}

func (b *Booking) MarkAsCompleted() {
	now := time.Now()
	b.CompletedAt = &now
	b.Status = BookingStatusCompleted
	b.PaymentStatus = PaymentStatusCompleted
}
