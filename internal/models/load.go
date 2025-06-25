package models

import "time"

// Load represents a shipment that needs to be transported
type Load struct {
	ID           string `json:"id"`
	ShipperID    string `json:"shipper_id"`
	ShipperName  string `json:"shipper_name"`
	ShipperPhone string `json:"shipper_phone"`

	// Route details
	FromCity    string  `json:"from_city"`
	ToCity      string  `json:"to_city"`
	PickupPoint string  `json:"pickup_point"`
	DropPoint   string  `json:"drop_point"`
	Distance    float64 `json:"distance"` // in km

	// Load details
	Material    string  `json:"material"`     // e.g., "Electronics", "Textiles"
	Weight      float64 `json:"weight"`       // in tons
	VehicleType string  `json:"vehicle_type"` // required vehicle type

	// Pricing
	Price        float64 `json:"price"`         // offered price
	PaymentTerms string  `json:"payment_terms"` // e.g., "Advance", "To-Pay", "POD"

	// Timing
	LoadingDate time.Time `json:"loading_date"`

	// Status
	Status string `json:"status"` // "available", "booked", "in-transit", "delivered"

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoadSearch parameters for searching loads
type LoadSearch struct {
	FromCity    string `json:"from_city"`
	ToCity      string `json:"to_city"`
	VehicleType string `json:"vehicle_type"`
	DateFrom    string `json:"date_from"`
}
