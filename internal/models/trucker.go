package models

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Trucker represents a truck driver in the system
type Trucker struct {
	// Using gorm.Model gives us ID (uint), CreatedAt, UpdatedAt, DeletedAt automatically
	gorm.Model

	// Keep your TruckerID as string for backward compatibility
	TruckerID          string     `json:"trucker_id" gorm:"uniqueIndex"`
	Name               string     `json:"name"`
	Phone              string     `json:"phone" gorm:"uniqueIndex"`      // WhatsApp number - unique
	AadhaarLast4       string     `json:"aadhaar_last4"`                 // Last 4 digits for privacy
	VehicleNo          string     `json:"vehicle_no" gorm:"uniqueIndex"` // Vehicle number should be unique
	VehicleType        string     `json:"vehicle_type"`                  // e.g., "32ft multi axle", "19ft truck"
	Capacity           float64    `json:"capacity"`                      // in tons
	Verified           bool       `json:"verified" gorm:"default:false"`
	Rating             float64    `json:"rating" gorm:"default:5.0"`
	TotalTrips         int        `json:"total_trips" gorm:"default:0"`
	CurrentCity        string     `json:"current_city"`
	Available          bool       `json:"available" gorm:"default:true"`
	IsActive           bool       `json:"is_active" gorm:"default:true"`
	IsSuspended        bool       `json:"is_suspended" gorm:"default:false"`
	DocumentExpiryDate *time.Time `json:"document_expiry_date"`
	PaidAt             *time.Time `json:"paid_at"` // For payment tracking

	// Note: CreatedAt and UpdatedAt are automatically handled by gorm.Model

	// Relationships (optional - add when you need them)
	// Bookings []Booking `json:"bookings,omitempty" gorm:"foreignKey:TruckerID;references:TruckerID"`
}

// BeforeCreate hook to auto-generate TruckerID and normalize data
func (t *Trucker) BeforeCreate(tx *gorm.DB) error {
	// Generate TruckerID if not set
	if t.TruckerID == "" {
		t.TruckerID = fmt.Sprintf("TR%d%03d", time.Now().Unix(), time.Now().Nanosecond()%1000)
	}

	// Normalize vehicle number (remove spaces, convert to uppercase)
	t.VehicleNo = strings.ToUpper(strings.ReplaceAll(t.VehicleNo, " ", ""))

	// Normalize phone number (ensure it starts with +91 if not already)
	if !strings.HasPrefix(t.Phone, "+") {
		t.Phone = "+91" + strings.TrimPrefix(t.Phone, "91")
	}

	// Set default rating if not set
	if t.Rating == 0 {
		t.Rating = 5.0
	}

	return nil
}

// TruckerRegistration is used for new trucker registration (KEEPING YOUR STRUCT AS IS)
type TruckerRegistration struct {
	Name        string  `json:"name" validate:"required"`
	Phone       string  `json:"phone" validate:"required"`
	VehicleNo   string  `json:"vehicle_no" validate:"required"`
	VehicleType string  `json:"vehicle_type" validate:"required"`
	Capacity    float64 `json:"capacity" validate:"required"`
}

// Helper methods for the Trucker model
func (t *Trucker) SetAvailable(available bool) {
	t.Available = available
}

func (t *Trucker) UpdateLocation(city string) {
	t.CurrentCity = city
}

func (t *Trucker) CompleteTrip(rating float64) {
	t.TotalTrips++
	// Update rating with weighted average
	if t.TotalTrips == 1 {
		t.Rating = rating
	} else {
		t.Rating = ((t.Rating * float64(t.TotalTrips-1)) + rating) / float64(t.TotalTrips)
	}
}

// IsEligibleForLoad checks if trucker can take a new load
func (t *Trucker) IsEligibleForLoad(requiredCapacity float64, requiredVehicleType string) bool {
	return t.Available &&
		t.Verified &&
		t.Capacity >= requiredCapacity &&
		(requiredVehicleType == "" || strings.Contains(strings.ToLower(t.VehicleType), strings.ToLower(requiredVehicleType)))
}
