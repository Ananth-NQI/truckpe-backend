package services

import (
	"fmt"
	"log"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// InteractiveTemplateService handles advanced WhatsApp UI templates
type InteractiveTemplateService struct {
	store         storage.Store
	twilioService *TwilioService
}

// NewInteractiveTemplateService creates a new interactive template service
func NewInteractiveTemplateService(store storage.Store, twilioService *TwilioService) *InteractiveTemplateService {
	return &InteractiveTemplateService{
		store:         store,
		twilioService: twilioService,
	}
}

// SendDeliveryCompleteTemplate sends an alternative delivery completion template
// This is used when the standard delivery_confirmation template fails or for A/B testing
func (i *InteractiveTemplateService) SendDeliveryCompleteTemplate(booking *models.Booking, trucker *models.Trucker, load *models.Load) error {
	templateService := NewTemplateService(i.twilioService)

	// Calculate earnings summary
	totalEarnings := 0.0
	bookings, _ := i.store.GetBookingsByTrucker(trucker.TruckerID)
	for _, b := range bookings {
		if b.Status == models.BookingStatusDelivered {
			totalEarnings += b.NetAmount
		}
	}

	params := map[string]string{
		"booking_id":     booking.BookingID,
		"route":          fmt.Sprintf("%s → %s", load.FromCity, load.ToCity),
		"earnings":       fmt.Sprintf("₹%.0f", booking.NetAmount),
		"total_earnings": fmt.Sprintf("₹%.0f", totalEarnings),
		"next_action":    "Search for new loads",
	}

	err := templateService.SendTemplate(trucker.Phone, "delivery_complete", params)
	if err != nil {
		log.Printf("Failed to send delivery_complete template: %v", err)
		return err
	}

	return nil
}

// SendBookingActionsTemplate sends interactive booking action buttons
func (i *InteractiveTemplateService) SendBookingActionsTemplate(booking *models.Booking, userPhone string) error {
	templateService := NewTemplateService(i.twilioService)

	// Determine available actions based on booking status
	actions := i.determineBookingActions(booking)

	params := map[string]string{
		"booking_id":    booking.BookingID,
		"status":        string(booking.Status),
		"action_1":      actions[0],
		"action_2":      actions[1],
		"action_3":      actions[2],
		"callback_data": fmt.Sprintf("booking_%s", booking.BookingID),
	}

	err := templateService.SendTemplate(userPhone, "booking_actions", params)
	if err != nil {
		log.Printf("Failed to send booking_actions template: %v", err)
		return err
	}

	return nil
}

// SendBookingActionsV2Template sends updated interactive booking actions with more options
func (i *InteractiveTemplateService) SendBookingActionsV2Template(booking *models.Booking, userPhone string) error {
	templateService := NewTemplateService(i.twilioService)

	// Get load details
	load, _ := i.store.GetLoad(booking.LoadID)

	// Enhanced actions based on context
	actions := i.determineEnhancedBookingActions(booking, load)

	params := map[string]string{
		"booking_id":     booking.BookingID,
		"route":          fmt.Sprintf("%s → %s", load.FromCity, load.ToCity),
		"status":         string(booking.Status),
		"primary_action": actions["primary"],
		"quick_action_1": actions["quick1"],
		"quick_action_2": actions["quick2"],
		"more_options":   actions["more"],
		"callback_data":  fmt.Sprintf("booking_v2_%s", booking.BookingID),
	}

	err := templateService.SendTemplate(userPhone, "booking_actions_v2", params)
	if err != nil {
		log.Printf("Failed to send booking_actions_v2 template: %v", err)
		return err
	}

	return nil
}

// SendPostLoadEasyTemplate sends simplified load posting interface
func (i *InteractiveTemplateService) SendPostLoadEasyTemplate(shipperPhone string) error {
	templateService := NewTemplateService(i.twilioService)

	// Get shipper details
	shipper, err := i.store.GetShipperByPhone(shipperPhone)
	if err != nil {
		return fmt.Errorf("shipper not found")
	}

	// Get common routes from shipper's history
	commonRoutes := i.getShipperCommonRoutes(shipper.ShipperID)

	// Get common materials
	commonMaterials := i.getCommonMaterials()

	params := map[string]string{
		"shipper_name":      shipper.CompanyName,
		"route_option_1":    commonRoutes[0],
		"route_option_2":    commonRoutes[1],
		"route_option_3":    commonRoutes[2],
		"material_option_1": commonMaterials[0],
		"material_option_2": commonMaterials[1],
		"material_option_3": commonMaterials[2],
		"callback_prefix":   "post_easy",
	}

	err = templateService.SendTemplate(shipperPhone, "post_load_easy", params)
	if err != nil {
		log.Printf("Failed to send post_load_easy template: %v", err)
		return err
	}

	return nil
}

// SendLoadSelectionTemplate sends load picker interface for truckers
func (i *InteractiveTemplateService) SendLoadSelectionTemplate(truckerPhone string, loads []*models.Load) error {
	if len(loads) == 0 {
		return fmt.Errorf("no loads to display")
	}

	templateService := NewTemplateService(i.twilioService)

	// Limit to 5 loads for the template
	displayLoads := loads
	if len(loads) > 5 {
		displayLoads = loads[:5]
	}

	// Format load options
	loadOptions := []string{}
	for _, load := range displayLoads {
		option := fmt.Sprintf("%s→%s, ₹%.0f",
			load.FromCity[:3], // Abbreviate city names
			load.ToCity[:3],
			load.Price)
		loadOptions = append(loadOptions, option)

		// Store full load details in a temporary map for callback handling
		// In production, this would be stored in session/cache
	}

	params := map[string]string{
		"header_text":     "Select a load to book",
		"load_count":      fmt.Sprintf("%d", len(displayLoads)),
		"load_option_1":   loadOptions[0],
		"load_id_1":       displayLoads[0].LoadID,
		"load_option_2":   "",
		"load_id_2":       "",
		"load_option_3":   "",
		"load_id_3":       "",
		"load_option_4":   "",
		"load_id_4":       "",
		"load_option_5":   "",
		"load_id_5":       "",
		"footer_text":     "Tap to view details and book",
		"callback_prefix": "select_load",
	}

	// Add additional load options if available
	for i := 1; i < len(loadOptions) && i < 5; i++ {
		params[fmt.Sprintf("load_option_%d", i+1)] = loadOptions[i]
		params[fmt.Sprintf("load_id_%d", i+1)] = displayLoads[i].LoadID
	}

	err := templateService.SendTemplate(truckerPhone, "load_selection", params)
	if err != nil {
		log.Printf("Failed to send load_selection template: %v", err)
		return err
	}

	return nil
}

// SendShipperOTPShareV1Template sends the original OTP sharing template (deprecated)
func (i *InteractiveTemplateService) SendShipperOTPShareV1Template(shipperPhone string, otp string, bookingID string) error {
	// This is the v1 template - we generally use v2, but keeping for backward compatibility
	templateService := NewTemplateService(i.twilioService)

	// Get booking details
	booking, _ := i.store.GetBooking(bookingID)
	trucker, _ := i.store.GetTruckerByID(booking.TruckerID)

	params := map[string]string{
		"otp":            otp,
		"booking_id":     bookingID,
		"trucker_name":   trucker.Name,
		"vehicle_number": trucker.VehicleNo,
		"validity":       "10 minutes",
	}

	err := templateService.SendTemplate(shipperPhone, "shipper_otp_share", params)
	if err != nil {
		log.Printf("Failed to send shipper_otp_share (v1) template: %v", err)
		return err
	}

	return nil
}

// SendPlatformUpdateTemplate sends platform update notifications
func (i *InteractiveTemplateService) SendPlatformUpdateTemplate(userPhone string, updateType string) error {
	templateService := NewTemplateService(i.twilioService)

	// Define update messages
	updates := map[string]map[string]string{
		"new_feature": {
			"title":       "New Feature Alert! 🎉",
			"description": "Voice message support is now live! Send voice notes for faster communication.",
			"action":      "Try it now",
		},
		"maintenance": {
			"title":       "Scheduled Maintenance 🔧",
			"description": "TruckPe will be under maintenance on Sunday 2-4 AM IST. Plan accordingly.",
			"action":      "Set reminder",
		},
		"policy_update": {
			"title":       "Policy Update 📋",
			"description": "Updated cancellation policy: 2 free cancellations per month. Check details.",
			"action":      "View policy",
		},
		"app_update": {
			"title":       "Update Available 📱",
			"description": "New TruckPe update with faster booking and bug fixes. Update now!",
			"action":      "Update app",
		},
	}

	update, exists := updates[updateType]
	if !exists {
		update = updates["new_feature"] // Default
	}

	params := map[string]string{
		"update_title":       update["title"],
		"update_description": update["description"],
		"action_text":        update["action"],
		"update_date":        time.Now().Format("02 Jan 2006"),
	}

	err := templateService.SendTemplate(userPhone, "platform_update", params)
	if err != nil {
		log.Printf("Failed to send platform_update template: %v", err)
		return err
	}

	return nil
}

// Helper methods

func (i *InteractiveTemplateService) determineBookingActions(booking *models.Booking) []string {
	actions := []string{"View Details", "Contact Support", "Cancel"}

	switch booking.Status {
	case models.BookingStatusConfirmed:
		actions[0] = "Mark Arrived"
		actions[1] = "View Route"
		actions[2] = "Report Issue"
	case models.BookingStatusInTransit:
		actions[0] = "Share Location"
		actions[1] = "Mark Delivered"
		actions[2] = "Report Delay"
	case models.BookingStatusDelivered:
		actions[0] = "View Earnings"
		actions[1] = "Download POD"
		actions[2] = "Rate Experience"
	}

	return actions
}

func (i *InteractiveTemplateService) determineEnhancedBookingActions(booking *models.Booking, load *models.Load) map[string]string {
	actions := make(map[string]string)

	switch booking.Status {
	case models.BookingStatusConfirmed:
		actions["primary"] = "Start Trip"
		actions["quick1"] = "📍 Navigate"
		actions["quick2"] = "📞 Call Shipper"
		actions["more"] = "More Options"
	case models.BookingStatusInTransit:
		actions["primary"] = "Update Status"
		actions["quick1"] = "📍 Share Live Location"
		actions["quick2"] = "⏰ Report Delay"
		actions["more"] = "Emergency SOS"
	case models.BookingStatusDelivered:
		actions["primary"] = "View Payment"
		actions["quick1"] = "📄 Get Receipt"
		actions["quick2"] = "⭐ Rate Trip"
		actions["more"] = "Report Issue"
	default:
		actions["primary"] = "View Details"
		actions["quick1"] = "📞 Support"
		actions["quick2"] = "❌ Cancel"
		actions["more"] = "Help"
	}

	return actions
}

func (i *InteractiveTemplateService) getShipperCommonRoutes(shipperID string) []string {
	// Get shipper's load history
	loads, err := i.store.GetLoadsByShipper(shipperID)
	if err != nil || len(loads) == 0 {
		// Return default popular routes
		return []string{
			"Delhi → Mumbai",
			"Mumbai → Bangalore",
			"Chennai → Hyderabad",
		}
	}

	// Count route frequency
	routeCount := make(map[string]int)
	for _, load := range loads {
		route := fmt.Sprintf("%s → %s", load.FromCity, load.ToCity)
		routeCount[route]++
	}

	// Sort by frequency
	type routeFreq struct {
		route string
		count int
	}

	routes := []routeFreq{}
	for route, count := range routeCount {
		routes = append(routes, routeFreq{route, count})
	}

	// Sort by count
	for i := 0; i < len(routes)-1; i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[j].count > routes[i].count {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}

	// Return top 3
	commonRoutes := []string{}
	for i := 0; i < 3 && i < len(routes); i++ {
		commonRoutes = append(commonRoutes, routes[i].route)
	}

	// Fill with defaults if needed
	defaultRoutes := []string{
		"Delhi → Mumbai",
		"Mumbai → Bangalore",
		"Chennai → Hyderabad",
	}

	for len(commonRoutes) < 3 {
		commonRoutes = append(commonRoutes, defaultRoutes[len(commonRoutes)])
	}

	return commonRoutes
}

func (i *InteractiveTemplateService) getCommonMaterials() []string {
	// In production, analyze from database
	// For now, return common material types
	return []string{
		"Electronics",
		"Textiles",
		"Auto Parts",
	}
}

// Broadcast methods for platform-wide updates

// BroadcastPlatformUpdate sends platform updates to all users
func (i *InteractiveTemplateService) BroadcastPlatformUpdate(updateType string) error {
	log.Printf("Broadcasting platform update: %s", updateType)

	// Get all active users
	truckers, _ := i.store.GetAllTruckers()
	shippers, _ := i.store.GetAllShippers()

	sentCount := 0
	failedCount := 0

	// Send to truckers
	for _, trucker := range truckers {
		if trucker.IsActive && !trucker.IsSuspended {
			err := i.SendPlatformUpdateTemplate(trucker.Phone, updateType)
			if err != nil {
				failedCount++
				log.Printf("Failed to send update to trucker %s: %v", trucker.Phone, err)
			} else {
				sentCount++
			}
		}
	}

	// Send to shippers
	for _, shipper := range shippers {
		err := i.SendPlatformUpdateTemplate(shipper.Phone, updateType)
		if err != nil {
			failedCount++
			log.Printf("Failed to send update to shipper %s: %v", shipper.Phone, err)
		} else {
			sentCount++
		}
	}

	log.Printf("Platform update broadcast complete. Sent: %d, Failed: %d", sentCount, failedCount)
	return nil
}

// Interactive callback handlers

// HandleLoadSelectionCallback processes load selection from interactive template
func (i *InteractiveTemplateService) HandleLoadSelectionCallback(userPhone string, selectedLoadID string) error {
	// Get trucker
	trucker, err := i.store.GetTruckerByPhone(userPhone)
	if err != nil {
		return fmt.Errorf("trucker not found")
	}

	// Create booking
	booking, err := i.store.CreateBooking(selectedLoadID, trucker.TruckerID)
	if err != nil {
		return fmt.Errorf("failed to create booking: %v", err)
	}
	_ = booking

	// Send booking confirmation
	whatsappService := NewWhatsAppService(i.store, i.twilioService)

	// Format message as if user typed "BOOK <LoadID>"
	message := fmt.Sprintf("BOOK %s", selectedLoadID)
	_, err = whatsappService.ProcessMessage(userPhone, message)
	if err != nil {
		return err
	}

	log.Printf("Load selection processed: Trucker %s booked load %s", trucker.TruckerID, selectedLoadID)
	return nil
}

// HandleBookingActionCallback processes booking action button clicks
func (i *InteractiveTemplateService) HandleBookingActionCallback(userPhone string, bookingID string, action string) error {
	// Map action to command
	commandMap := map[string]string{
		"Mark Arrived":   fmt.Sprintf("ARRIVED %s", bookingID),
		"Mark Delivered": fmt.Sprintf("DELIVER %s", bookingID),
		"Share Location": "LOCATION", // Special handling needed
		"Report Delay":   fmt.Sprintf("DELAY %s", bookingID),
		"Cancel":         fmt.Sprintf("CANCEL %s", bookingID),
		"Emergency SOS":  "EMERGENCY",
		"View Details":   fmt.Sprintf("TRACK %s", bookingID),
		"Start Trip":     fmt.Sprintf("ARRIVED %s", bookingID),
		"Update Status":  "STATUS",
	}

	command, exists := commandMap[action]
	if !exists {
		return fmt.Errorf("unknown action: %s", action)
	}

	// Process command through WhatsApp service
	whatsappService := NewWhatsAppService(i.store, i.twilioService)
	_, err := whatsappService.ProcessMessage(userPhone, command)
	if err != nil {
		return err
	}

	log.Printf("Booking action processed: %s for booking %s", action, bookingID)
	return nil
}

// TestInteractiveTemplates sends test messages for all interactive templates
func (i *InteractiveTemplateService) TestInteractiveTemplates(testPhone string) error {
	log.Printf("Testing interactive templates for %s", testPhone)

	// Test platform update
	err := i.SendPlatformUpdateTemplate(testPhone, "new_feature")
	if err != nil {
		log.Printf("Platform update template test failed: %v", err)
	}

	// Test post load easy (if shipper)
	shipper, _ := i.store.GetShipperByPhone(testPhone)
	if shipper != nil {
		err = i.SendPostLoadEasyTemplate(testPhone)
		if err != nil {
			log.Printf("Post load easy template test failed: %v", err)
		}
	}

	// Test load selection (if trucker)
	trucker, _ := i.store.GetTruckerByPhone(testPhone)
	if trucker != nil {
		// Get some test loads
		loads, _ := i.store.GetAvailableLoads()
		if len(loads) > 0 {
			err = i.SendLoadSelectionTemplate(testPhone, loads)
			if err != nil {
				log.Printf("Load selection template test failed: %v", err)
			}
		}
	}

	log.Printf("Interactive template tests completed for %s", testPhone)
	return nil
}

// Add these methods to your InteractiveTemplateService in internal/services/interactive_templates.go

// SendShipperLoadsTemplate sends an interactive template showing shipper's posted loads
func (s *InteractiveTemplateService) SendShipperLoadsTemplate(phone string, loads []*models.Load) error {
	if len(loads) == 0 {
		return s.sendPlainMessage(phone, "📋 *Your Loads*\n\nNo loads posted yet.\n\nType POST to create a new load.")
	}

	// Create interactive buttons for load management
	var buttons []map[string]interface{}

	// Show up to 3 loads as buttons
	for i, load := range loads {
		if i >= 3 {
			break
		}

		statusIcon := "🟢"
		if load.Status == "booked" {
			statusIcon = "🟡"
		} else if load.Status == "completed" {
			statusIcon = "✅"
		}

		buttonText := fmt.Sprintf("%s %s→%s", statusIcon, load.FromCity[:3], load.ToCity[:3])
		buttons = append(buttons, map[string]interface{}{
			"type": "reply",
			"reply": map[string]string{
				"id":    fmt.Sprintf("track_%s", load.LoadID),
				"title": buttonText,
			},
		})
	}

	// Add "Post New Load" button
	buttons = append(buttons, map[string]interface{}{
		"type": "reply",
		"reply": map[string]string{
			"id":    "post_new_load",
			"title": "📦 Post New Load",
		},
	})

	// Build header text
	headerText := fmt.Sprintf("📋 *Your Posted Loads*\n\nYou have %d active loads:", len(loads))

	// Build body with load details
	bodyText := ""
	for i, load := range loads {
		if i >= 5 {
			bodyText += fmt.Sprintf("\n... and %d more loads", len(loads)-5)
			break
		}

		statusText := "Available"
		if load.Status == "booked" {
			statusText = "Booked"
		} else if load.Status == "completed" {
			statusText = "Completed"
		}

		bodyText += fmt.Sprintf("\n\n*%s*\n📍 %s → %s\n💰 ₹%.0f\n📊 %s",
			load.LoadID,
			load.FromCity,
			load.ToCity,
			load.Price,
			statusText)
	}

	// Send interactive message
	template := WhatsAppTemplates["interactive_button_template"]
	if template.SID == "" {
		// Fallback to plain message if template not configured
		return s.sendPlainMessage(phone, headerText+bodyText)
	}

	contentVariables := map[string]string{
		"1": headerText,
		"2": bodyText,
		"3": "Select a load to track or post a new one",
	}

	// Create the interactive component
	persistentAction := map[string]interface{}{
		"buttons": buttons,
	}

	return s.twilioService.SendWhatsAppInteractiveTemplate(
		phone,
		template.SID,
		contentVariables,
		persistentAction,
	)
}

// SendTruckerStatusTemplate sends an interactive template showing trucker's bookings
func (s *InteractiveTemplateService) SendTruckerStatusTemplate(phone string, bookings []*models.Booking) error {
	if len(bookings) == 0 {
		return s.sendPlainMessage(phone, "📊 *Your Status*\n\nNo active bookings.\n\nSearch for loads: LOAD <from> <to>")
	}

	// Get trucker details
	trucker, err := s.store.GetTruckerByPhone(phone)
	if err != nil {
		return err
	}

	// Create interactive buttons based on booking status
	var buttons []map[string]interface{}
	activeBookingFound := false

	for _, booking := range bookings {
		// Only show buttons for active bookings
		if booking.Status == models.BookingStatusConfirmed || booking.Status == models.BookingStatusInTransit {
			activeBookingFound = true

			// Get load details
			load, _ := s.store.GetLoad(booking.LoadID)
			if load == nil {
				continue
			}

			var buttonText string
			var buttonID string

			if booking.Status == models.BookingStatusConfirmed && booking.PickedUpAt == nil {
				buttonText = fmt.Sprintf("📍 Arrived at %s", load.FromCity[:min(8, len(load.FromCity))])
				buttonID = fmt.Sprintf("arrived_%s", booking.BookingID)
			} else if booking.Status == models.BookingStatusInTransit {
				buttonText = fmt.Sprintf("📦 Deliver at %s", load.ToCity[:min(8, len(load.ToCity))])
				buttonID = fmt.Sprintf("deliver_%s", booking.BookingID)
			}

			if buttonText != "" {
				buttons = append(buttons, map[string]interface{}{
					"type": "reply",
					"reply": map[string]string{
						"id":    buttonID,
						"title": buttonText,
					},
				})
			}

			// Only show first active booking's actions
			break
		}
	}

	// Add search loads button if no active booking
	if !activeBookingFound {
		buttons = append(buttons, map[string]interface{}{
			"type": "reply",
			"reply": map[string]string{
				"id":    "search_loads",
				"title": "🔍 Search Loads",
			},
		})
	}

	// Add support button
	buttons = append(buttons, map[string]interface{}{
		"type": "reply",
		"reply": map[string]string{
			"id":    "contact_support",
			"title": "💬 Support",
		},
	})

	// Build header
	headerText := fmt.Sprintf("📊 *Your Bookings*\n👤 %s (%s)", trucker.Name, trucker.VehicleNo)

	// Build body with booking details
	bodyText := ""
	for i, booking := range bookings {
		if i >= 3 {
			bodyText += fmt.Sprintf("\n\n... and %d more bookings", len(bookings)-3)
			break
		}

		load, _ := s.store.GetLoad(booking.LoadID)
		if load == nil {
			continue
		}

		statusEmoji := "📦"
		if booking.Status == models.BookingStatusInTransit {
			statusEmoji = "🚛"
		} else if booking.Status == models.BookingStatusDelivered {
			statusEmoji = "✅"
		}

		bodyText += fmt.Sprintf("\n\n%s *%s*\n📍 %s → %s\n💰 ₹%.0f\n📊 %s",
			statusEmoji,
			booking.BookingID,
			load.FromCity,
			load.ToCity,
			booking.NetAmount,
			booking.Status)

		// Add action hint
		if booking.Status == models.BookingStatusConfirmed && booking.PickedUpAt == nil {
			bodyText += "\n👉 Ready for pickup"
		} else if booking.Status == models.BookingStatusInTransit {
			bodyText += "\n👉 In transit"
		}
	}

	// Send interactive message
	template := WhatsAppTemplates["interactive_button_template"]
	if template.SID == "" {
		// Fallback to plain message
		return s.sendPlainMessage(phone, headerText+bodyText)
	}

	contentVariables := map[string]string{
		"1": headerText,
		"2": bodyText,
		"3": "Select an action:",
	}

	persistentAction := map[string]interface{}{
		"buttons": buttons,
	}

	return s.twilioService.SendWhatsAppInteractiveTemplate(
		phone,
		template.SID,
		contentVariables,
		persistentAction,
	)
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *InteractiveTemplateService) sendPlainMessage(to string, message string) error {
	// For plain messages, we need to use a template since WhatsApp Business API
	// requires pre-approved templates for business messaging

	// Try to use a generic notification template if available
	if template, exists := WhatsAppTemplates["plain_text_message"]; exists && template.SID != "" {
		params := map[string]string{
			"1": message,
		}
		return s.twilioService.SendWhatsAppTemplate(to, template.SID, params)
	}

	// If no plain text template is available, log the message
	// In production, you should have a generic template for plain messages
	log.Printf("[PLAIN MESSAGE] To: %s, Message: %s", to, message)

	// Return nil to not block the flow
	// In production, you might want to return an error if no template is available
	return nil
}
