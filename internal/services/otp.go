package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/Ananth-NQI/truckpe-backend/internal/utils"
)

type OTPService struct {
	store storage.Store
}

func NewOTPService(store storage.Store) *OTPService {
	return &OTPService{store: store}
}

// GenerateSecureOTP generates a cryptographically secure 6-digit OTP
func (s *OTPService) GenerateSecureOTP() (string, error) {
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	// Ensure 6 digits with leading zeros
	return fmt.Sprintf("%06d", n.Int64()+1), nil
}

// CreateOTP creates a new OTP for the given purpose
func (s *OTPService) CreateOTP(phone, purpose, referenceID string) (*models.OTP, error) {
	// Use the secure OTP generation from utils
	code, err := utils.GenerateSecureOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}

	otp := &models.OTP{
		Phone:       phone,
		Code:        code,
		Purpose:     purpose,
		ReferenceID: referenceID,
		ExpiresAt:   time.Now().Add(10 * time.Minute), // 10 minute expiry
		IsUsed:      false,
		Attempts:    0,
	}

	return s.store.CreateOTP(otp)
}

// VerifyOTP verifies if the OTP is valid
func (s *OTPService) VerifyOTP(phone, code, purpose string) (bool, string, error) {
	otp, err := s.store.GetActiveOTP(phone, code, purpose)
	if err != nil {
		return false, "", err
	}

	// Check if expired
	if time.Now().After(otp.ExpiresAt) {
		return false, "", fmt.Errorf("OTP expired")
	}

	// Check if already used
	if otp.IsUsed {
		return false, "", fmt.Errorf("OTP already used")
	}

	// Check attempts
	otp.Attempts++
	if otp.Attempts > 3 {
		return false, "", fmt.Errorf("too many attempts")
	}

	// Mark as used
	now := time.Now()
	otp.VerifiedAt = &now
	otp.IsUsed = true

	err = s.store.UpdateOTP(otp)
	if err != nil {
		return false, "", err
	}

	return true, otp.ReferenceID, nil
}

// ResendOTP creates a new OTP for the same purpose (invalidates old ones)
func (s *OTPService) ResendOTP(phone, purpose, referenceID string) (*models.OTP, error) {
	// TODO: Mark old OTPs as used before creating new one
	return s.CreateOTP(phone, purpose, referenceID)
}
