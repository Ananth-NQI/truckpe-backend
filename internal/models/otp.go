package models

import (
	"time"

	"gorm.io/gorm"
)

type OTP struct {
	gorm.Model
	Phone       string    `gorm:"not null;index"`
	Code        string    `gorm:"not null"`
	Purpose     string    `gorm:"not null"` // "booking_pickup", "booking_delivery", "registration"
	ReferenceID string    `gorm:"index"`    // BookingID for booking OTPs
	ExpiresAt   time.Time `gorm:"not null"`
	VerifiedAt  *time.Time
	Attempts    int    `gorm:"default:0"`
	IsUsed      bool   `gorm:"default:false"`
	Metadata    string // JSON for additional data
}
