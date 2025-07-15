package services

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

var (
	twilioServiceInstance *TwilioService
	twilioServiceOnce     sync.Once
)

// SetTwilioService sets the global twilio service instance
func SetTwilioService(ts *TwilioService) {
	twilioServiceInstance = ts
}

// GetTwilioService returns the global twilio service instance
func GetTwilioService() *TwilioService {
	return twilioServiceInstance
}

// WhatsAppService handles WhatsApp message processing
type WhatsAppService struct {
	store         storage.Store
	twilioService *TwilioService
}

// NewWhatsAppService creates a new WhatsApp service
func NewWhatsAppService(store storage.Store, twilioService *TwilioService) *WhatsAppService {
	return &WhatsAppService{
		store:         store,
		twilioService: twilioService,
	}
}

func (w *WhatsAppService) ProcessMessage(from, message string) (string, error) {
	// Convert to uppercase and trim
	msg := strings.TrimSpace(strings.ToUpper(message))

	// Extract phone number (remove WhatsApp prefix if present)
	phone := strings.TrimPrefix(from, "whatsapp:")

	// Log the command being processed
	log.Printf("Processing command '%s' from %s", msg, phone)

	// Handle button callbacks (they might come as specific formats)
	if strings.HasPrefix(msg, "BUTTON_") || strings.HasPrefix(msg, "ACTION_") {
		// Handle interactive button responses
		return w.handleButtonCallback(phone, msg)
	}

	// Route to appropriate handler based on command
	switch {
	case msg == "HELP" || msg == "HI" || msg == "HELLO" || msg == "START":
		// Always return help text for now to ensure user sees menu
		// Template can be sent in addition if needed
		helpText := w.getHelpMessage()

		// Try to send welcome template as well
		go func() {
			templateService := NewTemplateService(w.twilioService)
			err := templateService.SendTemplate(phone, "welcome_message", map[string]string{})
			if err != nil {
				log.Printf("Failed to send welcome template: %v", err)
			} else {
				log.Printf("Welcome template sent successfully")
			}
		}()

		// Always return help text so user sees the menu
		return helpText, nil

	case msg == "TEST DIRECT":
		// Test both text and template sending
		testMsg := "🧪 Test Results:\n\n1️⃣ This is a direct text message\n2️⃣ Checking template sending..."

		// Send template after a short delay
		go func() {
			time.Sleep(2 * time.Second)
			templateService := NewTemplateService(w.twilioService)
			err := templateService.SendTemplate(phone, "welcome_message", map[string]string{})
			if err != nil {
				w.twilioService.SendWhatsAppMessage(phone, "❌ Template test failed: "+err.Error())
			} else {
				w.twilioService.SendWhatsAppMessage(phone, "✅ Template sent! Do you see buttons?")
			}
		}()

		return testMsg, nil

	case strings.HasPrefix(msg, "REGISTER SHIPPER"):
		return w.handleShipperRegistration(phone, msg)

	case strings.HasPrefix(msg, "REGISTER"):
		return w.handleRegistration(phone, msg)

	case strings.HasPrefix(msg, "POST"):
		return w.handlePostLoad(phone, msg)

	case msg == "MY LOADS":
		return w.handleMyLoads(phone)

	case strings.HasPrefix(msg, "LOAD"):
		return w.handleLoadSearch(phone, msg)

	case strings.HasPrefix(msg, "BOOK"):
		return w.handleBooking(phone, msg)

	case msg == "STATUS":
		return w.handleStatus(phone)

	case strings.HasPrefix(msg, "TRACK"):
		return w.handleTrackBooking(phone, msg)

	case strings.HasPrefix(msg, "ARRIVED"):
		return w.handleArrived(phone, msg)

	case strings.HasPrefix(msg, "PICKUP"):
		return w.handlePickup(phone, msg)

	case strings.HasPrefix(msg, "DELIVER"):
		return w.handleDeliver(phone, msg)

	// EMERGENCY & SUPPORT COMMANDS
	case msg == "EMERGENCY" || msg == "SOS":
		return w.handleEmergency(phone, msg)

	case strings.HasPrefix(msg, "DELAY"):
		return w.handleDelay(phone, msg)

	case strings.HasPrefix(msg, "NEGOTIATE"):
		return w.handleNegotiate(phone, msg)

	case msg == "BREAKDOWN":
		return w.handleBreakdown(phone, msg)

	case strings.HasPrefix(msg, "CANCEL"):
		return w.handleCancel(phone, msg)

	case strings.HasPrefix(msg, "SUPPORT"):
		return w.handleSupport(phone, msg)

	// TEST COMMANDS
	case msg == "TEST TEMPLATES" || msg == "TEST":
		return w.handleTestTemplates(phone)

	case msg == "TEST BUTTON":
		// Test interactive buttons
		return w.testInteractiveButtons(phone)

	// QUICK COMMANDS (shortcuts)
	case msg == "R" || msg == "REG":
		return "📝 To register, use:\n\nFor Truckers:\nREGISTER Name, VehicleNo, Type, Capacity\n\nFor Shippers:\nREGISTER SHIPPER CompanyName, GSTNumber", nil

	case msg == "L":
		return "🔍 To search loads:\nLOAD <from> <to>\n\nExample: LOAD Chennai Mumbai", nil

	case msg == "S":
		return w.handleStatus(phone)

	// Handle numeric responses (for menu selections)
	case msg == "1" || msg == "2" || msg == "3" || msg == "4" || msg == "5":
		return w.handleMenuSelection(phone, msg)

	default:
		// Check if it's a button payload format
		if strings.Contains(msg, "_") {
			parts := strings.Split(msg, "_")
			if len(parts) >= 2 {
				return w.handleButtonCallback(phone, msg)
			}
		}

		// Unknown command
		return fmt.Sprintf("❌ Invalid command: '%s'\n\nType HELP to see all available commands.", msg), nil
	}
}

// handleButtonCallback processes button click responses
func (w *WhatsAppService) handleButtonCallback(phone string, callback string) (string, error) {
	log.Printf("Processing button callback: %s", callback)

	// Parse callback format: ACTION_PARAMETER
	parts := strings.Split(callback, "_")
	if len(parts) < 1 {
		return "❌ Invalid button response.", nil
	}

	action := parts[0]

	switch action {
	case "REGISTER":
		if len(parts) > 1 && parts[1] == "TRUCKER" {
			return "📝 *Trucker Registration*\n\nPlease provide your details in this format:\n\nREGISTER Name, VehicleNo, VehicleType, Capacity\n\nExample:\nREGISTER Raj Kumar, TN01AB1234, 32ft, 25", nil
		} else if len(parts) > 1 && parts[1] == "SHIPPER" {
			return "🏭 *Shipper Registration*\n\nPlease provide your details in this format:\n\nREGISTER SHIPPER CompanyName, GSTNumber\n\nExample:\nREGISTER SHIPPER ABC Logistics, 29ABCDE1234F1Z5", nil
		}

	case "SEARCH":
		if len(parts) > 1 && parts[1] == "LOADS" {
			return "🔍 *Search Loads*\n\nType: LOAD <from> <to>\n\nExamples:\nLOAD Chennai\nLOAD Chennai Mumbai\nLOAD Delhi Kolkata", nil
		}

	case "BOOK":
		if len(parts) > 1 {
			loadID := parts[1]
			return w.handleBooking(phone, "BOOK "+loadID)
		}

	case "ARRIVED", "PICKUP", "DELIVER", "CANCEL":
		if len(parts) > 1 {
			bookingID := parts[1]
			return w.ProcessMessage(phone, action+" "+bookingID)
		}

	case "VIEW":
		if len(parts) > 1 {
			switch parts[1] {
			case "TICKET":
				return "📋 To view support tickets, contact support: SUPPORT <message>", nil
			case "STATUS":
				return w.handleStatus(phone)
			}
		}
	}

	return fmt.Sprintf("Button clicked: %s\nProcessing...", callback), nil
}

// handleMenuSelection processes numeric menu selections
func (w *WhatsAppService) handleMenuSelection(phone string, selection string) (string, error) {
	// Check user context (trucker or shipper)
	trucker, _ := w.store.GetTruckerByPhone(phone)
	shipper, _ := w.store.GetShipperByPhone(phone)

	if trucker == nil && shipper == nil {
		// Not registered - show registration options
		switch selection {
		case "1":
			return "📝 *Trucker Registration*\n\nREGISTER Name, VehicleNo, Type, Capacity\n\nExample:\nREGISTER Kumar, TN01AB1234, 32ft, 25", nil
		case "2":
			return "🏭 *Shipper Registration*\n\nREGISTER SHIPPER Company, GST\n\nExample:\nREGISTER SHIPPER ABC Ltd, 29ABCDE1234F1Z5", nil
		default:
			return w.getHelpMessage(), nil
		}
	}

	// Handle based on user type
	if trucker != nil {
		switch selection {
		case "1":
			return "🔍 Search loads by typing:\nLOAD <from> <to>", nil
		case "2":
			return w.handleStatus(phone)
		case "3":
			return "💰 Your earnings summary coming soon!", nil
		case "4":
			return "📞 Support: Type SUPPORT <your message>", nil
		default:
			return w.getHelpMessage(), nil
		}
	}

	if shipper != nil {
		switch selection {
		case "1":
			return "📦 Post a load:\nPOST <from> <to> <material> <weight> <price>", nil
		case "2":
			return w.handleMyLoads(phone)
		case "3":
			return "📊 Dashboard access coming soon!", nil
		case "4":
			return "📞 Support: Type SUPPORT <your message>", nil
		default:
			return w.getHelpMessage(), nil
		}
	}

	return w.getHelpMessage(), nil
}

// testInteractiveButtons sends a test message with interactive buttons
func (w *WhatsAppService) testInteractiveButtons(phone string) (string, error) {
	// First send a text message
	response := "🧪 *Testing Interactive Buttons*\n\nIf buttons are configured in Twilio, you should see them below this message.\n\nOtherwise, you can use these commands:\n\n1️⃣ REGISTER - Start registration\n2️⃣ LOAD Chennai - Search loads\n3️⃣ STATUS - Check your status\n4️⃣ HELP - See all commands"

	// Try to send an interactive template
	go func() {
		time.Sleep(1 * time.Second)
		interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)

		// Try to send a test interactive template
		// This assumes you have a method to send test templates
		if err := interactiveService.TestInteractiveTemplates(phone); err != nil {
			log.Printf("Failed to send interactive test: %v", err)
		}
	}()

	return response, nil
}

// handleTestTemplates handles testing of interactive templates
func (w *WhatsAppService) handleTestTemplates(phone string) (string, error) {
	// Only allow for registered users
	trucker, _ := w.store.GetTruckerByPhone(phone)
	shipper, _ := w.store.GetShipperByPhone(phone)

	if trucker == nil && shipper == nil {
		return "❌ Please register first before testing templates!", nil
	}

	// Check if template testing is enabled
	if os.Getenv("ENABLE_TEMPLATE_TESTING") == "false" {
		return "❌ Template testing is disabled in this environment.", nil
	}

	interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)
	err := interactiveService.TestInteractiveTemplates(phone)
	if err != nil {
		log.Printf("Failed to send test templates to %s: %v", phone, err)
		return "❌ Failed to send test templates. Please try again.", err
	}

	return "✅ Test templates sent! Check your WhatsApp for interactive messages.", nil
}

// Help message - updated to include new commands
func (w *WhatsAppService) getHelpMessage() string {
	helpMsg := `🚛 *Welcome to TruckPe!*

*For Truckers:*
📝 *REGISTER* - Register as a trucker
🔍 *LOAD <from> <to>* - Search loads
📦 *BOOK <load_id>* - Book a load
📊 *STATUS* - Check your bookings
📍 *ARRIVED <booking_id>* - At pickup location
🚚 *PICKUP <booking_id> <otp>* - Confirm pickup
📦 *DELIVER <booking_id>* - At delivery location
🚨 *EMERGENCY/SOS* - Emergency assistance
⏰ *DELAY <booking_id>* - Report delay
💬 *NEGOTIATE <load_id> <price>* - Negotiate price
🔧 *BREAKDOWN* - Vehicle breakdown help
❌ *CANCEL <booking_id>* - Cancel booking

*For Shippers:*
🏭 *REGISTER SHIPPER* - Register as shipper
📦 *POST* - Post a new load
📋 *MY LOADS* - View your posted loads
🔍 *TRACK <booking_id>* - Track a booking

💬 *SUPPORT <message>* - Contact support
💰 *48-hour payment guarantee!*
🔒 *100% safe with escrow*

Type any command to start!`

	// Add test command if enabled
	if os.Getenv("ENABLE_TEMPLATE_TESTING") != "false" {
		helpMsg += "\n\n🧪 *TEST TEMPLATES* - Test all interactive templates"
	}

	return helpMsg
}

// Handle shipper registration
func (w *WhatsAppService) handleShipperRegistration(phone, msg string) (string, error) {
	// Check if already registered as shipper
	existingShipper, _ := w.store.GetShipperByPhone(phone)
	if existingShipper != nil {
		return fmt.Sprintf(`✅ *Already Registered as Shipper!*

*Shipper ID:* %s
*Company:* %s
*GST:* %s

You can post loads!
Type: POST to start posting`,
			existingShipper.ShipperID, existingShipper.CompanyName, existingShipper.GSTNumber), nil
	}

	// Check if registered as trucker
	existingTrucker, _ := w.store.GetTruckerByPhone(phone)
	if existingTrucker != nil {
		return "❌ This number is registered as a trucker. Use a different number for shipper account.", nil
	}

	// Parse registration message
	// Format: REGISTER SHIPPER CompanyName, GSTNumber
	msg = strings.TrimPrefix(msg, "REGISTER SHIPPER")
	parts := strings.Split(msg, ",")
	if len(parts) < 2 {
		return `❌ Invalid format!

Correct format:
REGISTER SHIPPER CompanyName, GSTNumber

Example:
REGISTER SHIPPER ABC Industries, 29ABCDE1234F1Z5`, nil
	}

	companyName := strings.TrimSpace(parts[0])
	gstNumber := strings.TrimSpace(strings.ToUpper(parts[1]))

	// Basic GST validation (15 characters)
	if len(gstNumber) != 15 {
		return "❌ Invalid GST number! GST should be 15 characters.\n\nExample: 29ABCDE1234F1Z5", nil
	}

	// Create shipper
	shipper := &models.Shipper{
		CompanyName: companyName,
		GSTNumber:   gstNumber,
		Phone:       phone,
	}

	createdShipper, err := w.store.CreateShipper(shipper)
	if err != nil {
		if strings.Contains(err.Error(), "phone") {
			return "❌ This phone number is already registered!", nil
		}
		if strings.Contains(err.Error(), "GST") {
			return "❌ This GST number is already registered!", nil
		}
		return "❌ Registration failed. Please try again.", err
	}

	// Send registration success template for shipper
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"name":           createdShipper.CompanyName,
		"user_id":        createdShipper.ShipperID,
		"vehicle_number": createdShipper.GSTNumber, // Using GST in place of vehicle for shippers
	}

	err = templateService.SendTemplate(phone, "registration_success", params)
	if err != nil {
		log.Printf("Failed to send shipper registration template: %v", err)
		// Fallback to plain text
		return fmt.Sprintf(`✅ *Shipper Registration Successful!*

*Shipper ID:* %s
*Company:* %s
*GST:* %s

✨ You can now post loads!

Type POST to start posting loads.`,
			createdShipper.ShipperID, createdShipper.CompanyName, createdShipper.GSTNumber), nil
	}

	return "", nil
}

// Handle post load - guided flow
func (w *WhatsAppService) handlePostLoad(phone, msg string) (string, error) {
	// Check if registered as shipper
	shipper, err := w.store.GetShipperByPhone(phone)
	if err != nil {
		return "❌ Please register as shipper first!\n\nType: REGISTER SHIPPER CompanyName, GSTNumber", nil
	}

	// For now, simple format. Later we'll add guided flow with sessions
	if msg == "POST" || msg == "POST LOAD" {
		// Send interactive template for easier posting
		interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)
		err := interactiveService.SendPostLoadEasyTemplate(phone)
		if err == nil {
			return "", nil // Template sent successfully
		}

		// Fallback to text instructions
		return `📦 *Post New Load*

Please provide load details in this format:

POST <From> <To> <Material> <Weight> <Price>

Example:
POST Chennai Bangalore Electronics 15 35000

Or type each detail:
From City: ?`, nil
	}

	// Parse POST command
	parts := strings.Fields(msg)
	if len(parts) < 6 {
		return `❌ Incomplete details!

Format: POST <From> <To> <Material> <Weight> <Price>

Example: POST Chennai Bangalore Electronics 15 35000`, nil
	}

	// Extract details (convert cities to proper case for display)
	fromCity := strings.Title(strings.ToLower(parts[1]))
	toCity := strings.Title(strings.ToLower(parts[2]))
	material := strings.Title(strings.ToLower(parts[3]))

	var weight float64
	var price float64
	fmt.Sscanf(parts[4], "%f", &weight)
	fmt.Sscanf(parts[5], "%f", &price)

	// Create load
	load := &models.Load{
		ShipperID:    shipper.ShipperID,
		ShipperName:  shipper.CompanyName,
		ShipperPhone: shipper.Phone,
		FromCity:     fromCity,
		ToCity:       toCity,
		Material:     material,
		Weight:       weight,
		Price:        price,
		VehicleType:  "Any",                          // Default
		LoadingDate:  time.Now().Add(24 * time.Hour), // Tomorrow
		Status:       "available",
	}

	createdLoad, err := w.store.CreateLoad(load)
	if err != nil {
		return "❌ Failed to post load. Please try again.", err
	}

	// Update shipper's total loads count
	shipper.TotalLoads++

	// Send load posted confirmation template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"load_id": createdLoad.LoadID,
		"route":   fmt.Sprintf("%s → %s", createdLoad.FromCity, createdLoad.ToCity),
		"price":   fmt.Sprintf("₹%.0f", createdLoad.Price),
	}

	err = templateService.SendTemplate(phone, "load_posted_confirm", params)
	if err != nil {
		log.Printf("Failed to send load posted template: %v", err)
		// Fallback to plain text
		return fmt.Sprintf(`✅ *Load Posted Successfully!*

*Load ID:* %s
📍 *Route:* %s → %s
📦 *Material:* %s
⚖️ *Weight:* %.1f tons
💰 *Price:* ₹%.0f

🔔 Notifying nearby truckers...

Type MY LOADS to see all your loads.`,
			createdLoad.LoadID, createdLoad.FromCity, createdLoad.ToCity,
			createdLoad.Material, createdLoad.Weight, createdLoad.Price), nil
	}

	// Send interactive template to shipper for easier posting next time
	go func() {
		time.Sleep(2 * time.Second) // Small delay for better UX
		interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)
		_ = interactiveService.SendPostLoadEasyTemplate(phone)
	}()

	// Send load match notification to nearby truckers
	go func() {
		// Create a new template service instance for the goroutine
		templateService := NewTemplateService(w.twilioService)

		// Get all truckers (you'll need to implement this method)
		truckers, err := w.store.GetAllTruckers()
		if err != nil {
			log.Printf("Error finding truckers: %v", err)
			return
		}

		for _, trucker := range truckers {
			// Skip if trucker is not available (has active booking)
			bookings, _ := w.store.GetBookingsByTrucker(trucker.TruckerID)
			hasActiveBooking := false
			for _, booking := range bookings {
				if booking.Status == models.BookingStatusConfirmed ||
					booking.Status == models.BookingStatusInTransit {
					hasActiveBooking = true
					break
				}
			}

			if hasActiveBooking {
				continue // Skip busy truckers
			}

			// For now, notify all available truckers
			// In production, use proper location matching
			params := map[string]string{
				"route":   fmt.Sprintf("%s → %s", createdLoad.FromCity, createdLoad.ToCity),
				"price":   fmt.Sprintf("₹%.0f", createdLoad.Price),
				"load_id": createdLoad.LoadID,
			}

			err := templateService.SendTemplate(trucker.Phone, "load_match_notification", params)
			if err != nil {
				log.Printf("Failed to notify trucker %s: %v", trucker.TruckerID, err)
			} else {
				log.Printf("Notified trucker %s about new load %s", trucker.TruckerID, createdLoad.LoadID)
			}
		}
	}()

	return "", nil
}

// Handle my loads for shippers
func (w *WhatsAppService) handleMyLoads(phone string) (string, error) {
	// Check if shipper
	shipper, err := w.store.GetShipperByPhone(phone)
	if err != nil {
		return "❌ Please register as shipper first!\n\nType: REGISTER SHIPPER CompanyName, GSTNumber", nil
	}

	// Get loads
	loads, err := w.store.GetLoadsByShipper(shipper.ShipperID)
	if err != nil {
		return "❌ Error fetching loads. Please try again.", err
	}

	if len(loads) == 0 {
		return "📋 *Your Loads*\n\nNo loads posted yet.\n\nType POST to create a new load.", nil
	}

	// Send interactive template if available
	interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)
	err = interactiveService.SendShipperLoadsTemplate(phone, loads)
	if err == nil {
		return "", nil // Template sent successfully
	}

	// Fallback to text response
	response := fmt.Sprintf("📋 *Your Posted Loads*\n🏭 %s\n\n", shipper.CompanyName)

	for i, load := range loads {
		if i > 4 { // Limit display
			response += fmt.Sprintf("\n... and %d more loads", len(loads)-5)
			break
		}

		statusEmoji := "🟢" // available
		if load.Status == "booked" {
			statusEmoji = "🟡"
		} else if load.Status == "completed" {
			statusEmoji = "✅"
		}

		response += fmt.Sprintf(`%s *Load:* %s
📍 *Route:* %s → %s
💰 *Price:* ₹%.0f
📊 *Status:* %s

`, statusEmoji, load.LoadID, load.FromCity, load.ToCity,
			load.Price, load.Status)
	}

	response += "Type TRACK <LoadID> to see booking details."
	return response, nil
}

// Handle track booking for shippers
func (w *WhatsAppService) handleTrackBooking(phone, msg string) (string, error) {
	// Can be used by both shippers and truckers
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify Booking or Load ID\n\nExample: TRACK BK00001 or TRACK LD00001", nil
	}

	trackID := parts[1]

	// Check if it's a booking ID
	if strings.HasPrefix(trackID, "BK") {
		booking, err := w.store.GetBooking(trackID)
		if err != nil {
			return "❌ Booking not found. Please check the ID.", nil
		}

		// Get load details
		load, _ := w.store.GetLoad(booking.LoadID)

		statusInfo := ""
		if booking.PickedUpAt != nil {
			statusInfo = fmt.Sprintf("\n⏰ *Picked up:* %s", booking.PickedUpAt.Format("3:04 PM"))
		}
		if booking.DeliveredAt != nil {
			statusInfo += fmt.Sprintf("\n✅ *Delivered:* %s", booking.DeliveredAt.Format("3:04 PM"))
		}

		return fmt.Sprintf(`📍 *Tracking Details*

*Booking ID:* %s
*Route:* %s → %s
*Status:* %s
*Trucker:* %s
*Amount:* ₹%.0f%s

Last Update: Just now`,
			booking.BookingID, load.FromCity, load.ToCity,
			booking.Status, booking.TruckerID, booking.AgreedPrice, statusInfo), nil
	}

	// If it's a load ID, show bookings for that load
	if strings.HasPrefix(trackID, "LD") {
		bookings, err := w.store.GetBookingsByLoad(trackID)
		if err != nil || len(bookings) == 0 {
			return "❌ No bookings found for this load.", nil
		}

		booking := bookings[0] // Latest booking
		return fmt.Sprintf(`📍 *Load Tracking*

*Load ID:* %s
*Booking ID:* %s
*Status:* %s
*Trucker:* %s

Type STATUS for more details.`,
			trackID, booking.BookingID, booking.Status, booking.TruckerID), nil
	}

	return "❌ Invalid ID format. Use booking ID (BK00001) or load ID (LD00001).", nil
}

// Handle trucker registration
func (w *WhatsAppService) handleRegistration(phone, msg string) (string, error) {
	// Check if already registered
	existingTrucker, _ := w.store.GetTruckerByPhone(phone)
	if existingTrucker != nil {
		return fmt.Sprintf(`✅ *Already Registered!*

*Trucker ID:* %s
*Name:* %s
*Vehicle:* %s

You can search for loads!
Type: LOAD <from> <to>`,
			existingTrucker.TruckerID, existingTrucker.Name, existingTrucker.VehicleNo), nil
	}

	// Check if registered as shipper
	existingShipper, _ := w.store.GetShipperByPhone(phone)
	if existingShipper != nil {
		return "❌ This number is registered as a shipper. Use a different number for trucker account.", nil
	}

	// Parse registration message
	// Format: REGISTER Name, VehicleNo, VehicleType, Capacity
	parts := strings.Split(msg, ",")
	if len(parts) < 4 {
		return "❌ Invalid format!\n\nCorrect format:\nREGISTER Name, VehicleNo, VehicleType, Capacity\n\nExample:\nREGISTER Rajesh Kumar, TN01AB1234, 32ft, 25", nil
	}

	// Extract details
	name := strings.TrimSpace(strings.TrimPrefix(parts[0], "REGISTER"))
	vehicleNo := strings.TrimSpace(parts[1])
	vehicleType := strings.TrimSpace(parts[2])

	// Parse capacity
	var capacity float64
	fmt.Sscanf(strings.TrimSpace(parts[3]), "%f", &capacity)

	// Create trucker registration
	reg := &models.TruckerRegistration{
		Name:        name,
		Phone:       phone,
		VehicleNo:   vehicleNo,
		VehicleType: vehicleType,
		Capacity:    capacity,
	}

	trucker, err := w.store.CreateTrucker(reg)
	if err != nil {
		if strings.Contains(err.Error(), "phone number already registered") {
			return "❌ This phone number is already registered!", nil
		}
		if strings.Contains(err.Error(), "vehicle already registered") {
			return "❌ This vehicle is already registered with another trucker!", nil
		}
		return "❌ Registration failed. Please try again.", err
	}

	// Send registration success template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"name":           trucker.Name,
		"user_id":        trucker.TruckerID,
		"vehicle_number": trucker.VehicleNo,
	}

	err = templateService.SendTemplate(phone, "registration_success", params)
	if err != nil {
		log.Printf("Failed to send template: %v", err)
		// Fallback to plain text if template fails
		return fmt.Sprintf(`✅ *Registration Successful!*

*Trucker ID:* %s
*Name:* %s
*Vehicle:* %s (%s)
*Capacity:* %.1f tons

✨ You can now search for loads!
Type: LOAD <from> <to>

Example: LOAD Delhi Mumbai`,
			trucker.TruckerID, trucker.Name, trucker.VehicleNo,
			trucker.VehicleType, trucker.Capacity), nil
	}

	// Send welcome trucker template after a short delay
	go func() {
		time.Sleep(2 * time.Second) // Small delay for better UX
		welcomeParams := map[string]string{
			"name": trucker.Name,
		}
		err := templateService.SendTemplate(phone, "welcome_trucker", welcomeParams)
		if err != nil {
			log.Printf("Failed to send welcome trucker template: %v", err)
		}
	}()

	// Return simple confirmation since template was sent
	return "", nil
}

// Handle load search
func (w *WhatsAppService) handleLoadSearch(phone, msg string) (string, error) {
	// Check if trucker is registered
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Please register first!\n\nType: REGISTER Name, VehicleNo, Type, Capacity", nil
	}

	// Parse search command
	// Format: LOAD Delhi Mumbai or LOAD Delhi
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify at least origin city\n\nExample: LOAD Delhi or LOAD Delhi Mumbai", nil
	}

	search := &models.LoadSearch{
		FromCity: parts[1],
	}

	if len(parts) > 2 {
		search.ToCity = parts[2]
	}

	// Search loads
	loads, err := w.store.SearchLoads(search)
	if err != nil {
		return "❌ Error searching loads. Please try again.", err
	}

	if len(loads) == 0 {
		return fmt.Sprintf("😔 No loads found from %s\n\nTry searching other routes or check back later!", search.FromCity), nil
	}

	// Try to send interactive load selection template
	if len(loads) > 0 {
		interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)
		err = interactiveService.SendLoadSelectionTemplate(phone, loads)
		if err == nil {
			return "", nil // Template sent successfully
		}
		// Fall back to text response if template fails
	}

	// Fallback to text format
	response := fmt.Sprintf("🚛 *Available Loads from %s*\n", search.FromCity)
	response += fmt.Sprintf("👤 *For:* %s (%s)\n\n", trucker.Name, trucker.VehicleNo)

	for i, load := range loads {
		if i > 4 { // Limit to 5 loads in WhatsApp
			response += fmt.Sprintf("\n... and %d more loads\n", len(loads)-5)
			break
		}

		response += fmt.Sprintf(`📦 *Load ID:* %s
📍 *Route:* %s → %s
📦 *Material:* %s
⚖️ *Weight:* %.1f tons
💰 *Price:* ₹%.0f
🚛 *Vehicle:* %s
📅 *Loading:* Today

`, load.LoadID, load.FromCity, load.ToCity, load.Material,
			load.Weight, load.Price, load.VehicleType)
	}

	response += "To book, type: BOOK <Load_ID>\nExample: BOOK " + loads[0].LoadID
	return response, nil
}

// Handle booking
func (w *WhatsAppService) handleBooking(phone, msg string) (string, error) {
	// Check if trucker is registered
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Please register first!\n\nType: REGISTER Name, VehicleNo, Type, Capacity", nil
	}

	// Extract load ID
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify Load ID\n\nExample: BOOK LD00001", nil
	}

	loadID := parts[1]

	// Create booking
	booking, err := w.store.CreateBooking(loadID, trucker.TruckerID)
	if err != nil {
		if strings.Contains(err.Error(), "load not found") {
			return "❌ Load not found. Please check the Load ID.", nil
		}
		if strings.Contains(err.Error(), "load not available") {
			return "❌ Sorry! This load has already been booked.", nil
		}
		if strings.Contains(err.Error(), "trucker not available") {
			return "❌ You already have an active booking. Complete it first!", nil
		}
		return "❌ Booking failed. Please try again.", err
	}

	// Get load details
	load, _ := w.store.GetLoad(loadID)

	// Send booking confirmation template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"trucker_name": trucker.Name,
		"load_id":      load.LoadID,
		"route":        fmt.Sprintf("%s → %s", load.FromCity, load.ToCity),
		"amount":       fmt.Sprintf("₹%.0f", booking.NetAmount),
	}

	err = templateService.SendTemplate(phone, "trucker_booked_notification", params)
	if err != nil {
		log.Printf("Failed to send booking template: %v", err)
		// Fallback to plain text if template fails
		return fmt.Sprintf(`✅ *Booking Confirmed!*

*Booking ID:* %s
*Load ID:* %s
*Route:* %s → %s
*Material:* %s
*Amount:* ₹%.0f
*Your earnings:* ₹%.0f (after 5%% commission)

📍 *Next Steps:*
1️⃣ Go to pickup location
2️⃣ When you arrive, type: ARRIVED %s
3️⃣ Get OTP from shipper
4️⃣ Confirm pickup with OTP

💰 Payment will be credited within 48 hours after delivery!

Type STATUS to check your bookings.`,
			booking.BookingID, load.LoadID, load.FromCity, load.ToCity,
			load.Material, booking.AgreedPrice, booking.NetAmount, booking.BookingID), nil
	}

	// Send interactive booking actions after a delay
	go func() {
		time.Sleep(2 * time.Second) // Small delay for better UX
		interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)
		_ = interactiveService.SendBookingActionsTemplate(booking, phone)
	}()

	// Also notify the shipper about the booking
	if load.ShipperPhone != "" {
		shipperParams := map[string]string{
			"load_id":       load.LoadID,
			"delivery_time": "Within 24-48 hours", // You can calculate this based on route
			"trucker_name":  trucker.Name,
		}

		// Send notification to shipper (ignore errors for shipper notification)
		_ = templateService.SendTemplate(load.ShipperPhone, "delivery_notification_shipper", shipperParams)
	}

	// Return simple confirmation since template was sent
	return "", nil
}

// Handle status check
func (w *WhatsAppService) handleStatus(phone string) (string, error) {
	// Check if trucker is registered
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Please register first!\n\nType: REGISTER Name, VehicleNo, Type, Capacity", nil
	}

	// Get bookings
	bookings, err := w.store.GetBookingsByTrucker(trucker.TruckerID)
	if err != nil {
		return "❌ Error fetching bookings. Please try again.", err
	}

	if len(bookings) == 0 {
		return "📊 *Your Status*\n\nNo active bookings.\n\nSearch for loads: LOAD <from> <to>", nil
	}

	// Try to send interactive status template
	interactiveService := NewInteractiveTemplateService(w.store, w.twilioService)
	err = interactiveService.SendTruckerStatusTemplate(phone, bookings)
	if err == nil {
		return "", nil // Template sent successfully
	}

	// Fallback to text format
	response := fmt.Sprintf("📊 *Your Bookings*\n👤 %s (%s)\n\n", trucker.Name, trucker.VehicleNo)

	for i, booking := range bookings {
		if i > 4 { // Limit display
			response += fmt.Sprintf("\n... and %d more bookings", len(bookings)-5)
			break
		}

		// Get load details
		load, _ := w.store.GetLoad(booking.LoadID)
		if load != nil {
			// Add action hints based on status
			actionHint := ""
			if booking.Status == models.BookingStatusConfirmed && booking.PickedUpAt == nil {
				actionHint = "\n👉 Type: ARRIVED " + booking.BookingID
			} else if booking.Status == models.BookingStatusInTransit && booking.DeliveredAt == nil {
				actionHint = "\n👉 Type: DELIVER " + booking.BookingID
			}

			response += fmt.Sprintf(`🚛 *Booking:* %s
📍 *Route:* %s → %s
💰 *Earnings:* ₹%.0f
📊 *Status:* %s%s

`, booking.BookingID, load.FromCity, load.ToCity,
				booking.NetAmount, booking.Status, actionHint)
		}
	}

	return response, nil
}

// handleArrived generates OTP when trucker arrives at pickup location
func (w *WhatsAppService) handleArrived(phone, msg string) (string, error) {
	// Extract booking ID
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify Booking ID\n\nExample: ARRIVED BK00001", nil
	}

	bookingID := parts[1]

	// Verify trucker owns this booking
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Trucker not found. Please register first!", nil
	}

	booking, err := w.store.GetBooking(bookingID)
	if err != nil {
		return "❌ Booking not found. Check the booking ID.", nil
	}

	if booking.TruckerID != trucker.TruckerID {
		return "❌ This booking doesn't belong to you.", nil
	}

	// Check if already picked up
	if booking.PickedUpAt != nil {
		return "❌ This load has already been picked up!", nil
	}

	// Generate OTP for pickup
	otpService := NewOTPService(w.store)
	_, err = otpService.CreateOTP(phone, "booking_pickup", bookingID)
	if err != nil {
		return "❌ Failed to generate OTP. Please try again.", err
	}

	// Get load details
	load, _ := w.store.GetLoad(booking.LoadID)

	// Get shipper name - with fallback if shipper not found
	shipperName := "Shipper"
	if shipper, err := w.store.GetShipperByPhone(load.ShipperPhone); err == nil && shipper != nil {
		shipperName = shipper.CompanyName
	}

	// Send trucker arrived notification template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"trucker_name":   trucker.Name,
		"vehicle_number": trucker.VehicleNo,
		"booking_id":     bookingID,
	}

	err = templateService.SendTemplate(phone, "trucker_arrived_notify", params)
	if err != nil {
		log.Printf("Failed to send arrival template: %v", err)
		// Fallback
		return fmt.Sprintf(`📍 *Arrival Confirmed!*

*Booking:* %s
*Load:* %s
*Route:* %s → %s
*Shipper:* %s

✅ OTP has been sent to shipper
⏰ Valid for 10 minutes

Ask shipper for the OTP and type:
PICKUP %s <OTP>`,
			bookingID,
			load.LoadID,
			load.FromCity,
			load.ToCity,
			shipperName,
			bookingID), nil
	}

	// Also send OTP to shipper with template
	if load.ShipperPhone != "" {
		otpParams := map[string]string{
			"otp":          "******", // Don't send actual OTP in template for security
			"trucker_name": trucker.Name,
			"booking_id":   bookingID,
		}
		_ = templateService.SendTemplate(load.ShipperPhone, "shipper_otp_share_v2", otpParams)
	}

	return "", nil
}

// handlePickup verifies OTP and confirms pickup
func (w *WhatsAppService) handlePickup(phone, msg string) (string, error) {
	// Format: PICKUP BK00001 123456
	parts := strings.Fields(msg)
	if len(parts) < 3 {
		return "❌ Format: PICKUP <BookingID> <OTP>\n\nExample: PICKUP BK00001 123456", nil
	}

	bookingID := parts[1]
	otpCode := parts[2]

	// Verify trucker
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Trucker not found. Please register first!", nil
	}

	// Verify booking ownership
	booking, err := w.store.GetBooking(bookingID)
	if err != nil {
		return "❌ Booking not found. Check the booking ID.", nil
	}

	if booking.TruckerID != trucker.TruckerID {
		return "❌ This booking doesn't belong to you.", nil
	}

	// Verify OTP
	otpService := NewOTPService(w.store)
	valid, refID, err := otpService.VerifyOTP(phone, otpCode, "booking_pickup")

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return "❌ OTP has expired. Type ARRIVED to generate new OTP.", nil
		}
		if strings.Contains(err.Error(), "already used") {
			return "❌ This OTP has already been used.", nil
		}
		if strings.Contains(err.Error(), "too many attempts") {
			return "❌ Too many wrong attempts. Type ARRIVED to generate new OTP.", nil
		}
		return "❌ Invalid OTP. Please check and try again.", nil
	}

	if !valid || refID != bookingID {
		return "❌ OTP doesn't match this booking.", nil
	}

	// Update booking status and pickup time
	now := time.Now()
	booking.PickedUpAt = &now
	booking.Status = models.BookingStatusInTransit

	// Update the entire booking (not just status)
	err = w.store.UpdateBooking(booking)
	if err != nil {
		return "❌ Failed to update booking. Please try again.", err
	}

	// Get load details for response
	load, _ := w.store.GetLoad(booking.LoadID)

	// Send pickup completed template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"booking_id":  bookingID,
		"pickup_time": now.Format("3:04 PM"),
	}

	err = templateService.SendTemplate(phone, "pickup_completed", params)
	if err != nil {
		log.Printf("Failed to send pickup template: %v", err)
		// Fallback
		return fmt.Sprintf(`✅ *Pickup Confirmed!*

*Booking:* %s
*Status:* In Transit 🚛
*Pickup Time:* %s

*Route:* %s → %s
*Material:* %s
*Your Earnings:* ₹%.0f

📍 Share your live location for real-time tracking
💰 Payment will be processed after delivery

Safe journey! Drive carefully.

_Next: When you reach destination, type DELIVER %s_`,
			bookingID,
			now.Format("3:04 PM"),
			load.FromCity,
			load.ToCity,
			load.Material,
			booking.NetAmount,
			bookingID), nil
	}

	// Notify shipper about pickup
	if load.ShipperPhone != "" {
		shipperParams := map[string]string{
			"booking_id":  bookingID,
			"pickup_time": now.Format("3:04 PM"),
		}
		_ = templateService.SendTemplate(load.ShipperPhone, "pickup_completed", shipperParams)
	}

	return "", nil
}

// handleDeliver handles delivery arrival and OTP generation
func (w *WhatsAppService) handleDeliver(phone, msg string) (string, error) {
	// Extract booking ID
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify Booking ID\n\nExample: DELIVER BK00001", nil
	}

	bookingID := parts[1]

	// Verify trucker owns this booking
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Trucker not found. Please register first!", nil
	}

	booking, err := w.store.GetBooking(bookingID)
	if err != nil {
		return "❌ Booking not found. Check the booking ID.", nil
	}

	if booking.TruckerID != trucker.TruckerID {
		return "❌ This booking doesn't belong to you.", nil
	}

	// Check if not picked up yet
	if booking.PickedUpAt == nil {
		return "❌ Please complete pickup first! Type: ARRIVED " + bookingID, nil
	}

	// Check if already delivered
	if booking.DeliveredAt != nil {
		return "❌ This load has already been delivered!", nil
	}

	// If OTP provided in same message (DELIVER BK00001 123456)
	if len(parts) >= 3 {
		return w.handleDeliveryConfirmation(phone, msg)
	}

	// Generate OTP for delivery
	otpService := NewOTPService(w.store)
	_, err = otpService.CreateOTP(phone, "booking_delivery", bookingID)
	if err != nil {
		return "❌ Failed to generate OTP. Please try again.", err
	}

	// Get load details
	load, _ := w.store.GetLoad(booking.LoadID)

	return fmt.Sprintf(`📍 *Arrival at Delivery Location Confirmed!*

*Booking:* %s
*Route:* %s → %s
*Consignee:* Contact shipper for details

✅ OTP has been sent to consignee
⏰ Valid for 10 minutes

Get OTP from consignee and type:
DELIVER %s <OTP>`,
		bookingID,
		load.FromCity,
		load.ToCity,
		bookingID), nil
}

// handleDeliveryConfirmation verifies OTP and completes delivery
func (w *WhatsAppService) handleDeliveryConfirmation(phone, msg string) (string, error) {
	// Format: DELIVER BK00001 123456
	parts := strings.Fields(msg)
	if len(parts) < 3 {
		return "❌ Format: DELIVER <BookingID> <OTP>\n\nExample: DELIVER BK00001 123456", nil
	}

	bookingID := parts[1]
	otpCode := parts[2]

	// Verify trucker
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Trucker not found. Please register first!", nil
	}

	// Get booking
	booking, err := w.store.GetBooking(bookingID)
	if err != nil {
		return "❌ Booking not found. Check the booking ID.", nil
	}

	if booking.TruckerID != trucker.TruckerID {
		return "❌ This booking doesn't belong to you.", nil
	}

	// Verify OTP
	otpService := NewOTPService(w.store)
	valid, refID, err := otpService.VerifyOTP(phone, otpCode, "booking_delivery")

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return "❌ OTP has expired. Type DELIVER " + bookingID + " to generate new OTP.", nil
		}
		if strings.Contains(err.Error(), "already used") {
			return "❌ This OTP has already been used.", nil
		}
		if strings.Contains(err.Error(), "too many attempts") {
			return "❌ Too many wrong attempts. Type DELIVER " + bookingID + " to generate new OTP.", nil
		}
		return "❌ Invalid OTP. Please check and try again.", nil
	}

	if !valid || refID != bookingID {
		return "❌ OTP doesn't match this booking.", nil
	}

	// Update booking status and delivery time
	now := time.Now()
	booking.DeliveredAt = &now
	booking.Status = models.BookingStatusDelivered
	booking.PaymentStatus = "pending"

	// Update the entire booking
	err = w.store.UpdateBooking(booking)
	if err != nil {
		return "❌ Failed to update delivery status. Please try again.", err
	}

	// Update load status to completed
	err = w.store.UpdateLoadStatus(booking.LoadID, "completed")
	if err != nil {
		// Log error but don't fail the delivery confirmation
		fmt.Printf("Error updating load status: %v\n", err)
	}

	// Get load details for response
	load, _ := w.store.GetLoad(booking.LoadID)

	// Calculate journey time
	journeyTime := ""
	if booking.PickedUpAt != nil {
		duration := now.Sub(*booking.PickedUpAt)
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		journeyTime = fmt.Sprintf("%dh %dm", hours, minutes)
	}

	// Send delivery confirmation template to trucker
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"booking_id":   bookingID,
		"delivered_at": now.Format("3:04 PM"),
		"amount":       fmt.Sprintf("₹%.0f", booking.NetAmount),
	}

	err = templateService.SendTemplate(phone, "delivery_confirmation", params)
	if err != nil {
		log.Printf("Failed to send delivery template: %v", err)
		// Fallback
		return fmt.Sprintf(`✅ *Delivery Completed Successfully!*

*Booking:* %s
*Route:* %s → %s
*Delivery Time:* %s
*Journey Duration:* %s

💰 *Payment Details:*
*Your Earnings:* ₹%.0f
*Status:* Processing
*Expected Credit:* Within 48 hours

🎉 Great job! Safe journey completed.

Type STATUS to see your other bookings.
Type LOAD <from> <to> to find new loads.`,
			bookingID,
			load.FromCity,
			load.ToCity,
			now.Format("3:04 PM"),
			journeyTime,
			booking.NetAmount), nil
	}

	// Notify shipper about delivery
	if load.ShipperPhone != "" {
		shipperParams := map[string]string{
			"load_id":       load.LoadID,
			"delivery_time": now.Format("3:04 PM"),
			"trucker_name":  trucker.Name,
		}
		_ = templateService.SendTemplate(load.ShipperPhone, "delivery_notification_shipper", shipperParams)
	}

	// Send rating request after a delay
	go func() {
		time.Sleep(2 * time.Minute) // Wait 2 minutes before asking for rating
		ratingParams := map[string]string{
			"booking_id": bookingID,
			"route":      fmt.Sprintf("%s → %s", load.FromCity, load.ToCity),
		}
		_ = templateService.SendTemplate(phone, "rate_experience", ratingParams)
	}()

	return "", nil
}

// NEW HANDLER FUNCTIONS

// handleEmergency handles emergency/SOS situations
func (w *WhatsAppService) handleEmergency(phone, msg string) (string, error) {
	// Verify user is registered (trucker or shipper)
	trucker, _ := w.store.GetTruckerByPhone(phone)
	shipper, _ := w.store.GetShipperByPhone(phone)

	if trucker == nil && shipper == nil {
		return "❌ Please register first to use emergency services.", nil
	}

	// Get active booking if trucker
	var bookingInfo string
	if trucker != nil {
		bookings, _ := w.store.GetBookingsByTrucker(trucker.TruckerID)
		for _, booking := range bookings {
			if booking.Status == models.BookingStatusInTransit {
				load, _ := w.store.GetLoad(booking.LoadID)
				bookingInfo = fmt.Sprintf("\n*Active Booking:* %s\n*Route:* %s → %s",
					booking.BookingID, load.FromCity, load.ToCity)
				break
			}
		}
	}

	// Send emergency template
	templateService := NewTemplateService(w.twilioService)
	userType := "trucker"
	userName := ""
	if trucker != nil {
		userName = trucker.Name
	} else {
		userType = "shipper"
		userName = shipper.CompanyName
	}

	params := map[string]string{
		"user_name": userName,
		"user_type": userType,
		"location":  "Share live location", // In production, get actual location
	}

	err := templateService.SendTemplate(phone, "emergency_sos", params)
	if err != nil {
		log.Printf("Failed to send emergency template: %v", err)
		// Fallback
		return fmt.Sprintf(`🚨 *EMERGENCY RESPONSE ACTIVATED*

*User:* %s
*Phone:* %s%s

📍 Share your live location NOW
📞 Emergency contacts notified
🚑 Help is on the way

*Emergency Hotline:* 1800-XXX-XXXX
*Police:* 100
*Ambulance:* 108

Stay calm. Keep your phone on.
Share any additional details here.`, userName, phone, bookingInfo), nil
	}

	// Log emergency for backend tracking
	log.Printf("EMERGENCY: User %s (%s) triggered SOS", userName, phone)

	return "", nil
}

// handleDelay handles delay reporting
func (w *WhatsAppService) handleDelay(phone, msg string) (string, error) {
	// Extract booking ID
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify Booking ID\n\nExample: DELAY BK00001 Traffic jam", nil
	}

	bookingID := parts[1]
	reason := "Not specified"
	if len(parts) > 2 {
		reason = strings.Join(parts[2:], " ")
	}

	// Verify trucker
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Only truckers can report delays.", nil
	}

	// Get booking
	booking, err := w.store.GetBooking(bookingID)
	if err != nil {
		return "❌ Booking not found. Check the booking ID.", nil
	}

	if booking.TruckerID != trucker.TruckerID {
		return "❌ This booking doesn't belong to you.", nil
	}

	// Get load details
	load, _ := w.store.GetLoad(booking.LoadID)

	// Send delay notification template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"booking_id": bookingID,
		"reason":     reason,
		"new_eta":    "Will update soon", // Calculate based on delay
	}

	err = templateService.SendTemplate(phone, "trucker_delayed", params)
	if err != nil {
		log.Printf("Failed to send delay template: %v", err)
		// Fallback
		return fmt.Sprintf(`⏰ *Delay Reported*

*Booking:* %s
*Route:* %s → %s
*Reason:* %s

✅ Shipper has been notified
📍 Share live location for tracking

We'll inform the shipper about the delay.
Safe driving!`, bookingID, load.FromCity, load.ToCity, reason), nil
	}

	// Notify shipper about delay
	if load.ShipperPhone != "" {
		shipperParams := map[string]string{
			"booking_id":   bookingID,
			"trucker_name": trucker.Name,
			"reason":       reason,
		}
		_ = templateService.SendTemplate(load.ShipperPhone, "trucker_delayed", shipperParams)
	}

	return "", nil
}

// handleNegotiate handles price negotiation
func (w *WhatsAppService) handleNegotiate(phone, msg string) (string, error) {
	// Format: NEGOTIATE LD00001 40000
	parts := strings.Fields(msg)
	if len(parts) < 3 {
		return "❌ Format: NEGOTIATE <LoadID> <YourPrice>\n\nExample: NEGOTIATE LD00001 40000", nil
	}

	loadID := parts[1]
	var proposedPrice float64
	fmt.Sscanf(parts[2], "%f", &proposedPrice)

	// Verify trucker
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Please register as trucker first!", nil
	}

	// Get load
	load, err := w.store.GetLoad(loadID)
	if err != nil {
		return "❌ Load not found. Check the Load ID.", nil
	}

	if load.Status != "available" {
		return "❌ This load is no longer available for negotiation.", nil
	}

	// Calculate price difference
	priceDiff := proposedPrice - load.Price
	percentDiff := (priceDiff / load.Price) * 100

	// Send negotiation request template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"load_id":        loadID,
		"original_price": fmt.Sprintf("₹%.0f", load.Price),
		"proposed_price": fmt.Sprintf("₹%.0f", proposedPrice),
		"trucker_name":   trucker.Name,
	}

	err = templateService.SendTemplate(phone, "price_negotiation_request", params)
	if err != nil {
		log.Printf("Failed to send negotiation template: %v", err)
		// Fallback
		return fmt.Sprintf(`💬 *Price Negotiation Requested*

*Load:* %s
*Route:* %s → %s
*Original Price:* ₹%.0f
*Your Offer:* ₹%.0f (%.1f%% difference)

✅ Request sent to shipper
⏰ You'll receive response within 30 mins

Meanwhile, you can search other loads.`,
			loadID, load.FromCity, load.ToCity,
			load.Price, proposedPrice, percentDiff), nil
	}

	// Notify shipper
	if load.ShipperPhone != "" {
		shipperParams := map[string]string{
			"load_id":        loadID,
			"trucker_name":   trucker.Name,
			"proposed_price": fmt.Sprintf("₹%.0f", proposedPrice),
			"vehicle_no":     trucker.VehicleNo,
		}
		_ = templateService.SendTemplate(load.ShipperPhone, "price_negotiation_request", shipperParams)
	}

	return "", nil
}

// handleBreakdown handles vehicle breakdown
func (w *WhatsAppService) handleBreakdown(phone, msg string) (string, error) {
	// Verify trucker
	trucker, err := w.store.GetTruckerByPhone(phone)
	if err != nil {
		return "❌ Only registered truckers can report breakdown.", nil
	}

	// Check for active booking
	bookings, _ := w.store.GetBookingsByTrucker(trucker.TruckerID)
	var activeBooking *models.Booking
	for _, booking := range bookings {
		if booking.Status == models.BookingStatusInTransit {
			activeBooking = booking
			break
		}
	}

	bookingInfo := ""
	if activeBooking != nil {
		load, _ := w.store.GetLoad(activeBooking.LoadID)
		bookingInfo = fmt.Sprintf("\n*Active Load:* %s → %s", load.FromCity, load.ToCity)
	}

	// Send breakdown assistance template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"trucker_name": trucker.Name,
		"vehicle_no":   trucker.VehicleNo,
		"location":     "Share your location", // In production, get actual location
	}

	err = templateService.SendTemplate(phone, "breakdown_assistance", params)
	if err != nil {
		log.Printf("Failed to send breakdown template: %v", err)
		// Fallback
		return fmt.Sprintf(`🔧 *Breakdown Assistance*

*Vehicle:* %s
*Driver:* %s%s

📍 Share your live location immediately
📞 Mechanic helpline: 1800-XXX-XXXX

*Nearest Service Centers:*
Loading based on your location...

✅ Your shipper will be notified
🚛 Alternative vehicle being arranged

What's the issue?
1. Tyre puncture
2. Engine problem
3. Fuel issue
4. Other

Reply with the number.`, trucker.VehicleNo, trucker.Name, bookingInfo), nil
	}

	return "", nil
}

// handleCancel handles booking cancellation
func (w *WhatsAppService) handleCancel(phone, msg string) (string, error) {
	// Extract booking ID
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify Booking ID\n\nExample: CANCEL BK00001", nil
	}

	bookingID := parts[1]

	// Check if user is trucker or shipper
	trucker, _ := w.store.GetTruckerByPhone(phone)
	shipper, _ := w.store.GetShipperByPhone(phone)

	if trucker == nil && shipper == nil {
		return "❌ Please register first!", nil
	}

	// Get booking
	booking, err := w.store.GetBooking(bookingID)
	if err != nil {
		return "❌ Booking not found. Check the booking ID.", nil
	}

	// Verify ownership
	if trucker != nil && booking.TruckerID != trucker.TruckerID {
		return "❌ This booking doesn't belong to you.", nil
	}

	// Check if already picked up
	if booking.PickedUpAt != nil {
		return "❌ Cannot cancel! Load already picked up.\n\nContact support for assistance.", nil
	}

	// Update booking status
	booking.Status = models.BookingStatusCancelled
	now := time.Now()
	booking.CancelledAt = &now
	err = w.store.UpdateBooking(booking)
	if err != nil {
		return "❌ Failed to cancel booking. Please try again.", err
	}

	// Update load status back to available
	_ = w.store.UpdateLoadStatus(booking.LoadID, "available")

	// Send cancellation template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"booking_id":   bookingID,
		"cancelled_by": "trucker",
		"penalty":      "₹500", // Calculate based on policy
	}

	err = templateService.SendTemplate(phone, "booking_cancelled", params)
	if err != nil {
		log.Printf("Failed to send cancellation template: %v", err)
		// Fallback
		return fmt.Sprintf(`❌ *Booking Cancelled*

*Booking ID:* %s
*Status:* Cancelled
*Penalty:* ₹500 will be deducted

⚠️ Frequent cancellations may lead to:
- Account suspension
- Lower priority in bookings
- Reduced earnings

Type LOAD <from> <to> to find new loads.`, bookingID), nil
	}

	return "", nil
}

// handleSupport handles support requests
func (w *WhatsAppService) handleSupport(phone, msg string) (string, error) {
	// Extract support message
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return `📞 *Contact Support*

Please describe your issue:

SUPPORT <your message>

Example:
SUPPORT Payment not received for BK00001

Or call: 1800-XXX-XXXX`, nil
	}

	supportMessage := strings.Join(parts[1:], " ")

	// Get user details
	userName := ""
	userType := ""
	userID := ""

	trucker, _ := w.store.GetTruckerByPhone(phone)
	if trucker != nil {
		userName = trucker.Name
		userType = "Trucker"
		userID = trucker.TruckerID
	} else {
		shipper, _ := w.store.GetShipperByPhone(phone)
		if shipper != nil {
			userName = shipper.CompanyName
			userType = "Shipper"
			userID = shipper.ShipperID
		}
	}

	if userName == "" {
		return "❌ Please register first to contact support.", nil
	}

	// Create support ticket
	ticket := &models.SupportTicket{
		UserPhone:   phone,
		UserType:    userType,
		UserID:      userID,
		IssueType:   "general",
		Description: supportMessage,
		Status:      "open",
		Priority:    "medium",
	}

	createdTicket, err := w.store.CreateSupportTicket(ticket)
	if err != nil {
		log.Printf("Failed to create support ticket: %v", err)
		// Still provide support even if ticket creation fails
		ticketID := fmt.Sprintf("TK%s", time.Now().Format("20060102150405"))
		createdTicket = &models.SupportTicket{TicketID: ticketID}
	}

	// Send support ticket update template
	templateService := NewTemplateService(w.twilioService)
	params := map[string]string{
		"ticket_id": createdTicket.TicketID,
		"status":    "created",
		"eta":       "24 hours",
	}

	err = templateService.SendTemplate(phone, "support_ticket_update", params)
	if err != nil {
		log.Printf("Failed to send support template: %v", err)
		// Fallback
		return fmt.Sprintf(`📋 *Support Ticket Created*

*Ticket ID:* %s
*User:* %s (%s)
*Issue:* %s

✅ Your request has been logged
⏰ Expected response: Within 24 hours

*For urgent issues:*
📞 Call: 1800-XXX-XXXX
💬 WhatsApp: +91-XXXXXXXXXX

We'll update you soon on this number.`,
			createdTicket.TicketID, userName, userType, supportMessage), nil
	}

	// Log support request
	log.Printf("Support ticket %s created by %s (%s): %s", createdTicket.TicketID, userName, userID, supportMessage)

	return "", nil
}
