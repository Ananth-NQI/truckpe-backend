package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// GenerateSecureOTP generates a cryptographically secure 6-digit OTP
func GenerateSecureOTP() (string, error) {
	// Generate a random number between 0 and 999999
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}

	// Add 1 to avoid 0 and format with leading zeros to ensure 6 digits
	otp := n.Int64() + 1
	return fmt.Sprintf("%06d", otp), nil
}

// GenerateSecureID generates a secure random ID for bookings/loads
func GenerateSecureID(prefix string) string {
	// Generate a random 6-digit number
	max := big.NewInt(999999)
	n, _ := rand.Int(rand.Reader, max)

	// Use timestamp + random for uniqueness
	return fmt.Sprintf("%s%d%06d", prefix, time.Now().Unix(), n.Int64())
}
