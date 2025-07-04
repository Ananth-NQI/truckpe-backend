package models

import (
	"time"

	"gorm.io/gorm"
)

// WhatsAppSession stores temporary session data for WhatsApp conversations
type WhatsAppSession struct {
	gorm.Model
	PhoneNumber string    `json:"phone_number" gorm:"uniqueIndex"`
	LastCommand string    `json:"last_command"`
	Context     string    `json:"context"` // JSON string to store conversation context
	ExpiresAt   time.Time `json:"expires_at"`
}
