package jobs

import (
	"fmt"
	"log"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/services"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// NotificationJob handles scheduled notifications
type NotificationJob struct {
	store         storage.Store
	twilioService *services.TwilioService
	isRunning     bool
}

// NewNotificationJob creates a new notification job scheduler
func NewNotificationJob(store storage.Store, twilioService *services.TwilioService) *NotificationJob {
	return &NotificationJob{
		store:         store,
		twilioService: twilioService,
		isRunning:     false,
	}
}

// Start begins all scheduled notification jobs
func (n *NotificationJob) Start() {
	if n.isRunning {
		log.Println("Notification jobs already running")
		return
	}

	n.isRunning = true
	log.Println("Starting scheduled notification jobs...")

	// Start all scheduled jobs
	go n.scheduleWeeklySummary()
	go n.scheduleDocumentExpiryCheck()
	go n.scheduleMaintenanceReminders()
	go n.scheduleInactivityCheck()
	go n.scheduleBulkLoadAlerts()
	go n.scheduleMilestoneCheck()

	log.Println("All notification jobs started successfully")
}

// Stop halts all scheduled jobs
func (n *NotificationJob) Stop() {
	n.isRunning = false
	log.Println("Stopping scheduled notification jobs...")
}

// 1. WEEKLY SUMMARY - Runs every Sunday at 6 PM
func (n *NotificationJob) scheduleWeeklySummary() {
	for n.isRunning {
		now := time.Now()
		// Calculate next Sunday 6 PM
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 && now.Hour() >= 18 {
			daysUntilSunday = 7 // If it's Sunday after 6 PM, schedule for next Sunday
		}

		nextRun := time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 18, 0, 0, 0, now.Location())
		duration := nextRun.Sub(now)

		log.Printf("Next weekly summary scheduled in %v", duration)
		time.Sleep(duration)

		if !n.isRunning {
			break
		}

		n.sendWeeklySummaries()
	}
}

// sendWeeklySummaries sends weekly earning summaries to all active truckers
func (n *NotificationJob) sendWeeklySummaries() {
	log.Println("Sending weekly summaries...")

	templateService := services.NewTemplateService(n.twilioService)

	// Get all truckers
	truckers, err := n.store.GetAllTruckers()
	if err != nil {
		log.Printf("Error getting truckers for weekly summary: %v", err)
		return
	}

	sentCount := 0
	for _, trucker := range truckers {
		// Skip inactive truckers
		if trucker.IsSuspended || !trucker.IsActive {
			continue
		}

		// Get trucker stats for the week
		stats, err := n.store.GetTruckerStats(trucker.TruckerID)
		if err != nil {
			log.Printf("Error getting stats for trucker %s: %v", trucker.TruckerID, err)
			continue
		}

		// Get bookings from last 7 days
		startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		endDate := time.Now().Format("2006-01-02")
		bookings, err := n.store.GetCompletedBookingsInDateRange(startDate, endDate)
		if err != nil {
			log.Printf("Error getting bookings for date range: %v", err)
			continue
		}

		// Calculate weekly earnings
		weeklyEarnings := 0.0
		weeklyTrips := 0
		for _, booking := range bookings {
			if booking.TruckerID == trucker.TruckerID {
				weeklyEarnings += booking.NetAmount
				weeklyTrips++
			}
		}

		// Skip if no activity this week
		if weeklyTrips == 0 {
			continue
		}

		// Send weekly summary template
		params := map[string]string{
			"name":            trucker.Name,
			"weekly_earnings": fmt.Sprintf("₹%.0f", weeklyEarnings),
			"total_trips":     fmt.Sprintf("%d", weeklyTrips),
			"total_earnings":  fmt.Sprintf("₹%.0f", stats.TotalEarnings),
		}

		err = templateService.SendTemplate(trucker.Phone, "weekly_summary", params)
		if err != nil {
			log.Printf("Failed to send weekly summary to %s: %v", trucker.Phone, err)
			continue
		}

		sentCount++
	}

	log.Printf("Weekly summaries sent: %d", sentCount)
}

// 2. DOCUMENT EXPIRY REMINDER - Runs daily at 10 AM
func (n *NotificationJob) scheduleDocumentExpiryCheck() {
	for n.isRunning {
		now := time.Now()
		// Calculate next run at 10 AM
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		duration := nextRun.Sub(now)
		log.Printf("Next document expiry check scheduled in %v", duration)
		time.Sleep(duration)

		if !n.isRunning {
			break
		}

		n.checkDocumentExpiry()
	}
}

// checkDocumentExpiry checks for expiring documents and sends reminders
func (n *NotificationJob) checkDocumentExpiry() {
	log.Println("Checking for expiring documents...")

	templateService := services.NewTemplateService(n.twilioService)

	// Check documents expiring in next 30 days
	truckers, err := n.store.GetTruckersWithExpiringDocuments(30)
	if err != nil {
		log.Printf("Error getting truckers with expiring documents: %v", err)
		return
	}

	sentCount := 0
	for _, trucker := range truckers {
		if trucker.DocumentExpiryDate == nil {
			continue
		}

		daysUntilExpiry := int(time.Until(*trucker.DocumentExpiryDate).Hours() / 24)

		// Send reminder at 30, 15, 7, and 1 day before expiry
		if daysUntilExpiry == 30 || daysUntilExpiry == 15 || daysUntilExpiry == 7 || daysUntilExpiry == 1 {
			params := map[string]string{
				"name":           trucker.Name,
				"document_type":  "Vehicle Registration", // You might want to make this dynamic
				"expiry_date":    trucker.DocumentExpiryDate.Format("02 Jan 2006"),
				"days_remaining": fmt.Sprintf("%d", daysUntilExpiry),
			}

			err = templateService.SendTemplate(trucker.Phone, "document_expiry_reminder", params)
			if err != nil {
				log.Printf("Failed to send document expiry reminder to %s: %v", trucker.Phone, err)
				continue
			}

			sentCount++
		}
	}

	log.Printf("Document expiry reminders sent: %d", sentCount)
}

// 3. MAINTENANCE REMINDER - Runs daily at 8 AM
func (n *NotificationJob) scheduleMaintenanceReminders() {
	for n.isRunning {
		now := time.Now()
		// Calculate next run at 8 AM
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		duration := nextRun.Sub(now)
		log.Printf("Next maintenance reminder check scheduled in %v", duration)
		time.Sleep(duration)

		if !n.isRunning {
			break
		}

		n.sendMaintenanceReminders()
	}
}

// sendMaintenanceReminders sends vehicle maintenance reminders
func (n *NotificationJob) sendMaintenanceReminders() {
	log.Println("Sending maintenance reminders...")

	templateService := services.NewTemplateService(n.twilioService)

	// Get all active truckers
	truckers, err := n.store.GetAllTruckers()
	if err != nil {
		log.Printf("Error getting truckers for maintenance reminders: %v", err)
		return
	}

	sentCount := 0
	for _, trucker := range truckers {
		// Skip if suspended or inactive
		if trucker.IsSuspended || !trucker.IsActive {
			continue
		}

		// Get trucker's booking count
		bookings, _ := n.store.GetBookingsByTrucker(trucker.TruckerID)
		completedTrips := 0
		totalKm := 0.0

		for _, booking := range bookings {
			if booking.Status == models.BookingStatusDelivered {
				completedTrips++
				// Estimate distance based on route (in production, use actual tracking)
				totalKm += 200 // Placeholder - you'd calculate actual distance
			}
		}

		// Send reminder every 5000 km or 3 months (whichever comes first)
		// This is simplified - in production, track actual service dates
		if completedTrips > 0 && completedTrips%25 == 0 { // Roughly every 25 trips
			params := map[string]string{
				"name":           trucker.Name,
				"vehicle_number": trucker.VehicleNo,
				"service_type":   "Regular Service",
				"last_service":   "3 months ago", // Track this properly in production
			}

			err = templateService.SendTemplate(trucker.Phone, "maintenance_reminder", params)
			if err != nil {
				log.Printf("Failed to send maintenance reminder to %s: %v", trucker.Phone, err)
				continue
			}

			sentCount++
		}
	}

	log.Printf("Maintenance reminders sent: %d", sentCount)
}

// 4. INACTIVITY REMINDER - Runs daily at 2 PM
func (n *NotificationJob) scheduleInactivityCheck() {
	for n.isRunning {
		now := time.Now()
		// Calculate next run at 2 PM
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}

		duration := nextRun.Sub(now)
		log.Printf("Next inactivity check scheduled in %v", duration)
		time.Sleep(duration)

		if !n.isRunning {
			break
		}

		n.checkInactiveUsers()
	}
}

// checkInactiveUsers sends re-engagement messages to inactive users
func (n *NotificationJob) checkInactiveUsers() {
	log.Println("Checking for inactive users...")

	templateService := services.NewTemplateService(n.twilioService)

	// Check truckers inactive for 7 days
	inactiveTruckers, err := n.store.GetInactiveTruckers(7)
	if err != nil {
		log.Printf("Error getting inactive truckers: %v", err)
		return
	}

	sentCount := 0

	// Send reminders to truckers
	for _, trucker := range inactiveTruckers {
		params := map[string]string{
			"name":          trucker.Name,
			"days_inactive": "7",
			"last_earning":  "₹5,000", // Get actual last earning
		}

		err = templateService.SendTemplate(trucker.Phone, "inactivity_reminder", params)
		if err != nil {
			log.Printf("Failed to send inactivity reminder to trucker %s: %v", trucker.Phone, err)
			continue
		}

		sentCount++
	}

	// Check inactive shippers
	inactiveShippers, err := n.store.GetInactiveShippers(14) // 14 days for shippers
	if err != nil {
		log.Printf("Error getting inactive shippers: %v", err)
		return
	}

	for _, shipper := range inactiveShippers {
		params := map[string]string{
			"name":          shipper.CompanyName,
			"days_inactive": "14",
			"last_earning":  "", // Not applicable for shippers
		}

		err = templateService.SendTemplate(shipper.Phone, "inactivity_reminder", params)
		if err != nil {
			log.Printf("Failed to send inactivity reminder to shipper %s: %v", shipper.Phone, err)
			continue
		}

		sentCount++
	}

	log.Printf("Inactivity reminders sent: %d", sentCount)
}

// 5. BULK LOAD ALERT - Runs every hour
func (n *NotificationJob) scheduleBulkLoadAlerts() {
	for n.isRunning {
		time.Sleep(1 * time.Hour)

		if !n.isRunning {
			break
		}

		n.checkBulkLoadOpportunities()
	}
}

// checkBulkLoadOpportunities alerts truckers about multiple loads on their route
func (n *NotificationJob) checkBulkLoadOpportunities() {
	log.Println("Checking for bulk load opportunities...")

	templateService := services.NewTemplateService(n.twilioService)

	// Get all available loads
	loads, err := n.store.GetAvailableLoads()
	if err != nil {
		log.Printf("Error getting available loads: %v", err)
		return
	}

	// Group loads by route
	routeLoads := make(map[string][]*models.Load)
	for _, load := range loads {
		route := fmt.Sprintf("%s-%s", load.FromCity, load.ToCity)
		routeLoads[route] = append(routeLoads[route], load)
	}

	// Find routes with 3+ loads
	sentCount := 0
	for route, loads := range routeLoads {
		if len(loads) >= 3 {
			// Get available truckers
			truckers, _ := n.store.GetAvailableTruckers()

			for _, trucker := range truckers {
				// In production, check if trucker is near the pickup location
				// For now, notify all available truckers

				totalValue := 0.0
				for _, load := range loads {
					totalValue += load.Price
				}

				params := map[string]string{
					"route":       route,
					"load_count":  fmt.Sprintf("%d", len(loads)),
					"total_value": fmt.Sprintf("₹%.0f", totalValue),
				}

				err = templateService.SendTemplate(trucker.Phone, "bulk_load_alert", params)
				if err != nil {
					log.Printf("Failed to send bulk load alert to %s: %v", trucker.Phone, err)
					continue
				}

				sentCount++
			}
		}
	}

	log.Printf("Bulk load alerts sent: %d", sentCount)
}

// 6. MILESTONE ACHIEVEMENT - Checked after each delivery
func (n *NotificationJob) scheduleMilestoneCheck() {
	// This is event-driven rather than scheduled
	// Called from delivery completion handler
}

// CheckMilestones checks and sends milestone achievements
func (n *NotificationJob) CheckMilestones(truckerID string) {
	log.Printf("Checking milestones for trucker %s", truckerID)

	templateService := services.NewTemplateService(n.twilioService)

	// Get trucker stats
	stats, err := n.store.GetTruckerStats(truckerID)
	if err != nil {
		log.Printf("Error getting trucker stats: %v", err)
		return
	}

	// Get trucker details
	trucker, err := n.store.GetTruckerByID(truckerID)
	if err != nil {
		log.Printf("Error getting trucker: %v", err)
		return
	}

	// Define milestones
	milestones := map[int]string{
		1:   "First Delivery",
		10:  "Rising Star",
		25:  "Reliable Partner",
		50:  "Half Century",
		100: "Century Champion",
		250: "TruckPe Elite",
		500: "TruckPe Legend",
	}

	// Check if completed trips matches any milestone
	if milestone, exists := milestones[stats.CompletedTrips]; exists {
		params := map[string]string{
			"name":           trucker.Name,
			"milestone":      milestone,
			"trip_count":     fmt.Sprintf("%d", stats.CompletedTrips),
			"total_earnings": fmt.Sprintf("₹%.0f", stats.TotalEarnings),
		}

		err = templateService.SendTemplate(trucker.Phone, "milestone_achievement", params)
		if err != nil {
			log.Printf("Failed to send milestone achievement: %v", err)
			return
		}

		log.Printf("Milestone achievement sent: %s achieved %s", trucker.Name, milestone)
	}
}

// 7. REFERRAL PROGRAM - Sent periodically to top performers
func (n *NotificationJob) SendReferralInvites() {
	log.Println("Sending referral program invites...")

	templateService := services.NewTemplateService(n.twilioService)

	// Get top performing truckers
	truckers, err := n.store.GetAllTruckers()
	if err != nil {
		log.Printf("Error getting truckers for referral program: %v", err)
		return
	}

	sentCount := 0
	for _, trucker := range truckers {
		stats, _ := n.store.GetTruckerStats(trucker.TruckerID)

		// Send to truckers with 10+ completed trips
		if stats != nil && stats.CompletedTrips >= 10 {
			params := map[string]string{
				"name":          trucker.Name,
				"referral_code": fmt.Sprintf("TP%s", trucker.TruckerID[2:6]), // Simple referral code
				"bonus_amount":  "₹500",
			}

			err = templateService.SendTemplate(trucker.Phone, "referral_program", params)
			if err != nil {
				log.Printf("Failed to send referral invite to %s: %v", trucker.Phone, err)
				continue
			}

			sentCount++
		}
	}

	log.Printf("Referral invites sent: %d", sentCount)
}

// 8. FESTIVAL GREETING - Called on specific dates
func (n *NotificationJob) SendFestivalGreetings(festival string) {
	log.Printf("Sending %s greetings...", festival)

	templateService := services.NewTemplateService(n.twilioService)

	// Get all active users (truckers and shippers)
	truckers, _ := n.store.GetAllTruckers()
	shippers, _ := n.store.GetAllShippers()

	sentCount := 0

	// Send to truckers
	for _, trucker := range truckers {
		if trucker.IsActive && !trucker.IsSuspended {
			params := map[string]string{
				"name":     trucker.Name,
				"festival": festival,
			}

			err := templateService.SendTemplate(trucker.Phone, "festival_greeting", params)
			if err != nil {
				log.Printf("Failed to send festival greeting to trucker %s: %v", trucker.Phone, err)
				continue
			}

			sentCount++
		}
	}

	// Send to shippers
	for _, shipper := range shippers {
		params := map[string]string{
			"name":     shipper.CompanyName,
			"festival": festival,
		}

		err := templateService.SendTemplate(shipper.Phone, "festival_greeting", params)
		if err != nil {
			log.Printf("Failed to send festival greeting to shipper %s: %v", shipper.Phone, err)
			continue
		}

		sentCount++
	}

	log.Printf("%s greetings sent: %d", festival, sentCount)
}

// ScheduleFestivalGreetings sets up festival greeting schedule
func (n *NotificationJob) ScheduleFestivalGreetings() {
	// Define festival dates for the year
	festivals := map[string]time.Time{
		"Diwali":           time.Date(2024, 11, 1, 6, 0, 0, 0, time.UTC),
		"Holi":             time.Date(2024, 3, 25, 6, 0, 0, 0, time.UTC),
		"Dussehra":         time.Date(2024, 10, 12, 6, 0, 0, 0, time.UTC),
		"Eid":              time.Date(2024, 4, 11, 6, 0, 0, 0, time.UTC),
		"Christmas":        time.Date(2024, 12, 25, 6, 0, 0, 0, time.UTC),
		"New Year":         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		"Republic Day":     time.Date(2025, 1, 26, 6, 0, 0, 0, time.UTC),
		"Independence Day": time.Date(2024, 8, 15, 6, 0, 0, 0, time.UTC),
	}

	for festival, date := range festivals {
		go func(f string, d time.Time) {
			duration := time.Until(d)
			if duration > 0 {
				log.Printf("%s greetings scheduled in %v", f, duration)
				time.Sleep(duration)

				if n.isRunning {
					n.SendFestivalGreetings(f)
				}
			}
		}(festival, date)
	}
}
