package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"sort"

	"github.com/gofiber/fiber/v2"
)

// ValidateTwilioSignature validates that the webhook request is from Twilio
func ValidateTwilioSignature() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Twilio signature from header
		twilioSignature := c.Get("X-Twilio-Signature")
		if twilioSignature == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Missing Twilio signature",
			})
		}

		// Get auth token from environment
		authToken := os.Getenv("TWILIO_AUTH_TOKEN")
		if authToken == "" {
			// Log error but don't expose to client
			fmt.Println("ERROR: TWILIO_AUTH_TOKEN not set")
			return c.Status(500).JSON(fiber.Map{
				"error": "Server configuration error",
			})
		}

		// Build the URL (adjust based on your deployment)
		fullURL := getFullURL(c)

		// Get all form parameters
		formParams := make(map[string]string)
		c.Request().PostArgs().VisitAll(func(key, value []byte) {
			formParams[string(key)] = string(value)
		})

		// Calculate expected signature
		expectedSignature := calculateTwilioSignature(authToken, fullURL, formParams)

		// Compare signatures
		if twilioSignature != expectedSignature {
			return c.Status(401).JSON(fiber.Map{
				"error": "Invalid signature",
			})
		}

		return c.Next()
	}
}

// getFullURL constructs the full URL for the request
func getFullURL(c *fiber.Ctx) string {
	protocol := "https"
	if c.Protocol() == "http" {
		protocol = "http"
	}

	// For production on Cloud Run
	host := c.Hostname()
	path := c.Path()

	return fmt.Sprintf("%s://%s%s", protocol, host, path)
}

// calculateTwilioSignature calculates the expected signature
func calculateTwilioSignature(authToken, url string, params map[string]string) string {
	// Sort parameters by key
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the data string
	data := url
	for _, k := range keys {
		data += k + params[k]
	}

	// Calculate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(authToken))
	h.Write([]byte(data))

	// Return base64 encoded
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
