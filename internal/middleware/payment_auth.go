package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// ValidatePaymentSignature validates payment webhook signatures (Razorpay)
func ValidatePaymentSignature() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: Implement Razorpay signature validation
		// For now, just pass through
		return c.Next()
	}
}
