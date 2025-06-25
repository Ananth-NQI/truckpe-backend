package models

import "time"

// Trucker represents a truck driver in the system
type Trucker struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`         // WhatsApp number
	AadhaarLast4 string    `json:"aadhaar_last4"` // Last 4 digits for privacy
	VehicleNo    string    `json:"vehicle_no"`
	VehicleType  string    `json:"vehicle_type"` // e.g., "32ft multi axle", "19ft truck"
	Capacity     float64   `json:"capacity"`     // in tons
	Verified     bool      `json:"verified"`
	Rating       float64   `json:"rating"`
	TotalTrips   int       `json:"total_trips"`
	CurrentCity  string    `json:"current_city"`
	Available    bool      `json:"available"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TruckerRegistration is used for new trucker registration
type TruckerRegistration struct {
	Name        string  `json:"name" validate:"required"`
	Phone       string  `json:"phone" validate:"required"`
	VehicleNo   string  `json:"vehicle_no" validate:"required"`
	VehicleType string  `json:"vehicle_type" validate:"required"`
	Capacity    float64 `json:"capacity" validate:"required"`
}
