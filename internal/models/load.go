package models

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Load represents a shipment that needs to be transported
type Load struct {
	// Using gorm.Model gives us ID (uint), CreatedAt, UpdatedAt, DeletedAt automatically
	gorm.Model

	// Keep your LoadID as string for backward compatibility
	LoadID       string `json:"load_id" gorm:"uniqueIndex"`
	ShipperID    string `json:"shipper_id" gorm:"index"`
	ShipperName  string `json:"shipper_name"`
	ShipperPhone string `json:"shipper_phone" gorm:"index"`

	// Route details
	FromCity    string  `json:"from_city" gorm:"index"` // Index for faster search
	ToCity      string  `json:"to_city" gorm:"index"`   // Index for faster search
	PickupPoint string  `json:"pickup_point"`
	DropPoint   string  `json:"drop_point"`
	Distance    float64 `json:"distance"` // in km

	// Load details
	Material    string  `json:"material"`                  // e.g., "Electronics", "Textiles"
	Weight      float64 `json:"weight"`                    // in tons
	VehicleType string  `json:"vehicle_type" gorm:"index"` // required vehicle type - indexed for search

	// Pricing
	Price        float64 `json:"price"`         // offered price
	PaymentTerms string  `json:"payment_terms"` // e.g., "Advance", "To-Pay", "POD"

	// Timing
	LoadingDate time.Time `json:"loading_date" gorm:"index"` // Index for date-based searches

	// Status
	Status string `json:"status" gorm:"default:available;index"` // "available", "booked", "in-transit", "delivered"

	// Note: CreatedAt and UpdatedAt are automatically handled by gorm.Model

	// Relationships (optional - add when you need them)
	// Bookings []Booking `json:"bookings,omitempty" gorm:"foreignKey:LoadID;references:LoadID"`
}

// BeforeCreate hook to auto-generate LoadID and normalize data
func (l *Load) BeforeCreate(tx *gorm.DB) error {
	// Generate LoadID if not set
	if l.LoadID == "" {
		l.LoadID = fmt.Sprintf("LD%d%03d", time.Now().Unix(), time.Now().Nanosecond()%1000)
	}

	// Normalize city names (Title case)
	l.FromCity = strings.Title(strings.ToLower(strings.TrimSpace(l.FromCity)))
	l.ToCity = strings.Title(strings.ToLower(strings.TrimSpace(l.ToCity)))

	// Normalize phone number (ensure it starts with +91 if not already)
	if l.ShipperPhone != "" && !strings.HasPrefix(l.ShipperPhone, "+") {
		l.ShipperPhone = "+91" + strings.TrimPrefix(l.ShipperPhone, "91")
	}

	// Set default status if not set
	if l.Status == "" {
		l.Status = "available"
	}

	return nil
}

// LoadSearch parameters for searching loads (KEEPING YOUR STRUCT AS IS)
type LoadSearch struct {
	FromCity    string `json:"from_city"`
	ToCity      string `json:"to_city"`
	VehicleType string `json:"vehicle_type"`
	DateFrom    string `json:"date_from"`
}

// Load Status constants
const (
	LoadStatusAvailable = "available"
	LoadStatusBooked    = "booked"
	LoadStatusInTransit = "in-transit"
	LoadStatusDelivered = "delivered"
)

// Payment Terms constants
const (
	PaymentTermsAdvance = "Advance"
	PaymentTermsToPay   = "To-Pay"
	PaymentTermsPOD     = "POD"
)

// Helper methods for the Load model
func (l *Load) IsAvailable() bool {
	return l.Status == LoadStatusAvailable
}

func (l *Load) Book() {
	l.Status = LoadStatusBooked
}

func (l *Load) StartTransit() {
	l.Status = LoadStatusInTransit
}

func (l *Load) MarkDelivered() {
	l.Status = LoadStatusDelivered
}

// CalculateRate returns price per ton
func (l *Load) CalculateRate() float64 {
	if l.Weight == 0 {
		return 0
	}
	return l.Price / l.Weight
}

// CalculateRatePerKm returns price per kilometer
func (l *Load) CalculateRatePerKm() float64 {
	if l.Distance == 0 {
		return 0
	}
	return l.Price / l.Distance
}

// MatchesSearch checks if load matches search criteria
func (l *Load) MatchesSearch(search LoadSearch) bool {
	matches := true

	if search.FromCity != "" {
		matches = matches && strings.EqualFold(l.FromCity, search.FromCity)
	}

	if search.ToCity != "" {
		matches = matches && strings.EqualFold(l.ToCity, search.ToCity)
	}

	if search.VehicleType != "" {
		matches = matches && strings.Contains(strings.ToLower(l.VehicleType), strings.ToLower(search.VehicleType))
	}

	if search.DateFrom != "" {
		if date, err := time.Parse("2006-01-02", search.DateFrom); err == nil {
			matches = matches && !l.LoadingDate.Before(date)
		}
	}

	return matches
}
