package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// AdminHandler handles admin operations
type AdminHandler struct {
	store         storage.Store
	twilioService *services.TwilioService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(store storage.Store, twilioService *services.TwilioService) *AdminHandler {
	return &AdminHandler{
		store:         store,
		twilioService: twilioService,
	}
}

// GetPendingVerifications gets all pending verifications
func (h *AdminHandler) GetPendingVerifications(c *fiber.Ctx) error {
	verifications, err := h.store.GetPendingVerifications()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch pending verifications",
		})
	}

	return c.JSON(fiber.Map{
		"success":       true,
		"verifications": verifications,
		"count":         len(verifications),
	})
}

// UpdateVerification approves or rejects a verification
func (h *AdminHandler) UpdateVerification(c *fiber.Ctx) error {
	verificationID := c.Params("verificationID")

	var req struct {
		Status     string `json:"status"` // "approved" or "rejected"
		AdminNotes string `json:"admin_notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate status
	if req.Status != "approved" && req.Status != "rejected" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status must be 'approved' or 'rejected'",
		})
	}

	// Get verification details
	verification, err := h.store.GetVerification(verificationID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Verification not found",
		})
	}

	// Update verification status
	err = h.store.UpdateVerificationStatus(verificationID, req.Status, req.AdminNotes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update verification",
		})
	}

	// Get user details based on user type
	var userPhone string
	var userName string

	if verification.UserType == "trucker" {
		trucker, err := h.store.GetTruckerByID(verification.UserID)
		if err == nil {
			userPhone = trucker.Phone
			userName = trucker.Name
		}
	} else if verification.UserType == "shipper" {
		shipper, err := h.store.GetShipperByID(verification.UserID)
		if err == nil {
			userPhone = shipper.Phone
			userName = shipper.CompanyName
		}
	}

	// Send appropriate template based on status
	templateService := services.NewTemplateService(h.twilioService)

	if req.Status == "approved" {
		// Send verification approved template
		params := map[string]string{
			"name":          userName,
			"document_type": verification.DocumentType,
			"user_type":     verification.UserType,
		}

		err = templateService.SendTemplate(userPhone, "verification_approved", params)
		if err != nil {
			log.Printf("Failed to send verification approved template: %v", err)
		}

		log.Printf("Verification %s approved for %s (%s)", verificationID, userName, verification.UserID)

	} else if req.Status == "rejected" {
		// Send verification rejected template
		params := map[string]string{
			"name":          userName,
			"document_type": verification.DocumentType,
			"reason":        req.AdminNotes,
		}

		err = templateService.SendTemplate(userPhone, "verification_rejected", params)
		if err != nil {
			log.Printf("Failed to send verification rejected template: %v", err)
		}

		log.Printf("Verification %s rejected for %s (%s)", verificationID, userName, verification.UserID)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Verification %s successfully", req.Status),
		"verification": fiber.Map{
			"id":     verificationID,
			"status": req.Status,
		},
	})
}

// SuspendAccount suspends a trucker or shipper account
func (h *AdminHandler) SuspendAccount(c *fiber.Ctx) error {
	var req struct {
		UserType string `json:"user_type"` // "trucker" or "shipper"
		UserID   string `json:"user_id"`
		Reason   string `json:"reason"`
		Duration int    `json:"duration_days"` // 0 for permanent
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate user type
	if req.UserType != "trucker" && req.UserType != "shipper" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User type must be 'trucker' or 'shipper'",
		})
	}

	// Suspend account
	err := h.store.SuspendAccount(req.UserType, req.UserID, req.Reason)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to suspend account",
		})
	}

	// Get user details for notification
	var userPhone string
	var userName string

	if req.UserType == "trucker" {
		trucker, err := h.store.GetTruckerByID(req.UserID)
		if err == nil {
			userPhone = trucker.Phone
			userName = trucker.Name
		}
	} else {
		shipper, err := h.store.GetShipperByID(req.UserID)
		if err == nil {
			userPhone = shipper.Phone
			userName = shipper.CompanyName
		}
	}

	// Send account suspended template
	templateService := services.NewTemplateService(h.twilioService)

	suspensionType := "temporary"
	durationText := fmt.Sprintf("%d days", req.Duration)
	if req.Duration == 0 {
		suspensionType = "permanent"
		durationText = "permanently"
	}

	params := map[string]string{
		"name":            userName,
		"reason":          req.Reason,
		"suspension_type": suspensionType,
		"duration":        durationText,
		"support_contact": "1800-XXX-XXXX",
	}

	err = templateService.SendTemplate(userPhone, "account_suspended", params)
	if err != nil {
		log.Printf("Failed to send account suspended template: %v", err)
	}

	log.Printf("Account suspended: %s %s for %s", req.UserType, req.UserID, req.Reason)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Account suspended successfully",
		"suspension": fiber.Map{
			"user_type": req.UserType,
			"user_id":   req.UserID,
			"duration":  req.Duration,
			"reason":    req.Reason,
		},
	})
}

// ReactivateAccount reactivates a suspended account
func (h *AdminHandler) ReactivateAccount(c *fiber.Ctx) error {
	var req struct {
		UserType string `json:"user_type"`
		UserID   string `json:"user_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Reactivate account
	err := h.store.ReactivateAccount(req.UserType, req.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reactivate account",
		})
	}

	// Get user details for notification
	var userPhone string
	var userName string

	if req.UserType == "trucker" {
		trucker, err := h.store.GetTruckerByID(req.UserID)
		if err == nil {
			userPhone = trucker.Phone
			userName = trucker.Name

			// Send welcome back message
			templateService := services.NewTemplateService(h.twilioService)
			params := map[string]string{
				"name": userName,
			}
			_ = templateService.SendTemplate(userPhone, "welcome_trucker", params)
		}
	} else {
		shipper, err := h.store.GetShipperByID(req.UserID)
		if err == nil {
			userPhone = shipper.Phone
			userName = shipper.CompanyName
		}
	}

	log.Printf("Account reactivated: %s %s", req.UserType, req.UserID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Account reactivated successfully",
		"user": fiber.Map{
			"type": req.UserType,
			"id":   req.UserID,
			"name": userName,
		},
	})
}

// ExpireLoad manually expires a load
func (h *AdminHandler) ExpireLoad(c *fiber.Ctx) error {
	loadID := c.Params("loadID")

	// Get load details
	load, err := h.store.GetLoad(loadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Load not found",
		})
	}

	// Check if already expired or completed
	if load.Status == "expired" || load.Status == "completed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Load is already %s", load.Status),
		})
	}

	// Update load status to expired
	err = h.store.UpdateLoadStatus(loadID, "expired")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to expire load",
		})
	}

	// Get shipper details
	shipper, err := h.store.GetShipper(load.ShipperID)
	if err == nil && shipper != nil {
		// Send load expired notification template
		templateService := services.NewTemplateService(h.twilioService)
		params := map[string]string{
			"load_id": loadID,
			"route":   fmt.Sprintf("%s → %s", load.FromCity, load.ToCity),
			"reason":  "Manual expiry by admin",
		}

		err = templateService.SendTemplate(shipper.Phone, "load_expired_notification", params)
		if err != nil {
			log.Printf("Failed to send load expired template: %v", err)
		}
	}

	log.Printf("Load %s manually expired by admin", loadID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Load expired successfully",
		"load": fiber.Map{
			"id":     loadID,
			"status": "expired",
		},
	})
}

// GetExpiredLoads gets all expired loads
func (h *AdminHandler) GetExpiredLoads(c *fiber.Ctx) error {
	loads, err := h.store.GetLoadsByStatus("expired")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch expired loads",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"loads":   loads,
		"count":   len(loads),
	})
}

// GetPlatformOverview gets platform statistics
func (h *AdminHandler) GetPlatformOverview(c *fiber.Ctx) error {
	// Get various statistics
	truckers, _ := h.store.GetAllTruckers()
	shippers, _ := h.store.GetAllShippers()
	activeBookings, _ := h.store.GetActiveBookings()

	// Calculate active users
	activeTruckers := 0
	for _, t := range truckers {
		if t.IsActive && !t.IsSuspended {
			activeTruckers++
		}
	}

	activeShippers := 0
	for _, s := range shippers {
		if s.Active { // Actually use s to check if active
			activeShippers++
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"overview": fiber.Map{
			"total_truckers":  len(truckers),
			"active_truckers": activeTruckers,
			"total_shippers":  len(shippers),
			"active_shippers": activeShippers,
			"active_bookings": len(activeBookings),
			"platform_status": "operational",
			"last_updated":    time.Now(),
		},
	})
}

// GetRevenueStats gets revenue statistics
func (h *AdminHandler) GetRevenueStats(c *fiber.Ctx) error {
	// Get date range from query params
	startDate := c.Query("start_date", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDate := c.Query("end_date", time.Now().Format("2006-01-02"))

	// Get completed bookings in date range
	bookings, err := h.store.GetCompletedBookingsInDateRange(startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch revenue data",
		})
	}

	// Calculate revenue stats
	totalRevenue := 0.0
	platformCommission := 0.0
	truckerEarnings := 0.0

	for _, booking := range bookings {
		totalRevenue += booking.AgreedPrice
		commission := booking.AgreedPrice - booking.NetAmount
		platformCommission += commission
		truckerEarnings += booking.NetAmount
	}

	return c.JSON(fiber.Map{
		"success": true,
		"revenue": fiber.Map{
			"period": fiber.Map{
				"start": startDate,
				"end":   endDate,
			},
			"total_revenue":       totalRevenue,
			"platform_commission": platformCommission,
			"trucker_earnings":    truckerEarnings,
			"total_bookings":      len(bookings),
			"average_booking":     totalRevenue / float64(len(bookings)),
		},
	})
}

// TriggerVerificationPending sends verification pending notifications
func (h *AdminHandler) TriggerVerificationPending(userType, userID string) error {
	// Get user details
	var userPhone string
	var userName string

	if userType == "trucker" {
		trucker, err := h.store.GetTruckerByID(userID)
		if err != nil {
			return err
		}
		userPhone = trucker.Phone
		userName = trucker.Name
	} else {
		shipper, err := h.store.GetShipperByID(userID)
		if err != nil {
			return err
		}
		userPhone = shipper.Phone
		userName = shipper.CompanyName
	}

	// Send verification pending template
	templateService := services.NewTemplateService(h.twilioService)
	params := map[string]string{
		"name":          userName,
		"document_type": "KYC Documents",
		"expected_time": "24-48 hours",
	}

	err := templateService.SendTemplate(userPhone, "verification_pending", params)
	if err != nil {
		log.Printf("Failed to send verification pending template: %v", err)
		return err
	}

	log.Printf("Verification pending notification sent to %s", userName)
	return nil
}

// AutoExpireLoads automatically expires old loads (can be called by a cron job)
func (h *AdminHandler) AutoExpireLoads() error {
	// Get all available loads
	loads, err := h.store.GetAvailableLoads()
	if err != nil {
		return err
	}

	expiredCount := 0
	templateService := services.NewTemplateService(h.twilioService)

	for _, load := range loads {
		// Check if load is older than 7 days
		if time.Since(load.CreatedAt) > 7*24*time.Hour {
			// Expire the load
			err := h.store.UpdateLoadStatus(load.LoadID, "expired")
			if err != nil {
				log.Printf("Failed to expire load %s: %v", load.LoadID, err)
				continue
			}

			// Notify shipper
			shipper, err := h.store.GetShipper(load.ShipperID)
			if err == nil && shipper != nil {
				params := map[string]string{
					"load_id": load.LoadID,
					"route":   fmt.Sprintf("%s → %s", load.FromCity, load.ToCity),
					"reason":  "Load expired after 7 days",
				}

				_ = templateService.SendTemplate(shipper.Phone, "load_expired_notification", params)
			}

			expiredCount++
		}
	}

	log.Printf("Auto-expired %d loads", expiredCount)
	return nil
}
