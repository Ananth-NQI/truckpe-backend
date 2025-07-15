package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Verification struct {
	gorm.Model
	VerificationID string     `json:"verification_id" gorm:"uniqueIndex"`
	UserID         string     `json:"user_id" gorm:"index"`
	UserType       string     `json:"user_type"`     // "trucker" or "shipper"
	DocumentType   string     `json:"document_type"` // "RC", "DL", "Aadhaar", "PAN", "GST"
	DocumentURL    string     `json:"document_url"`
	Status         string     `json:"status" gorm:"default:pending"` // "pending", "approved", "rejected"
	AdminNotes     string     `json:"admin_notes"`
	VerifiedBy     string     `json:"verified_by"`
	VerifiedAt     *time.Time `json:"verified_at"`
}

func (v *Verification) BeforeCreate(tx *gorm.DB) error {
	if v.VerificationID == "" {
		v.VerificationID = fmt.Sprintf("VER%d", time.Now().Unix())
	}
	return nil
}
