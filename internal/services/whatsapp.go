package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// WhatsAppService handles WhatsApp message processing
type WhatsAppService struct {
	store storage.Store
}

// NewWhatsAppService creates a new WhatsApp service
func NewWhatsAppService(store storage.Store) *WhatsAppService {
	return &WhatsAppService{
		store: store,
	}
}

// ProcessMessage processes incoming WhatsApp messages
func (w *WhatsAppService) ProcessMessage(from, message string) (string, error) {
	// Convert to uppercase and trim
	msg := strings.TrimSpace(strings.ToUpper(message))

	// Extract phone number (remove WhatsApp prefix)
	phone := strings.TrimPrefix(from, "whatsapp:")

	// Route to appropriate handler based on command
	switch {
	case msg == "HELP" || msg == "HI" || msg == "HELLO":
		return w.getHelpMessage(), nil

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

	default:
		return "❌ Invalid command. Type HELP to see available commands.", nil
	}
}

// Help message - updated to include shipper commands
func (w *WhatsAppService) getHelpMessage() string {
	return `🚛 *Welcome to TruckPe!*

*For Truckers:*
📝 *REGISTER* - Register as a trucker
🔍 *LOAD <from> <to>* - Search loads
📦 *BOOK <load_id>* - Book a load
📊 *STATUS* - Check your bookings

*For Shippers:*
🏭 *REGISTER SHIPPER* - Register as shipper
📦 *POST* - Post a new load
📋 *MY LOADS* - View your posted loads
🔍 *TRACK <booking_id>* - Track a booking

💰 *48-hour payment guarantee!*
🔒 *100% safe with escrow*

Type any command to start!`
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

	return fmt.Sprintf(`✅ *Shipper Registration Successful!*

*Shipper ID:* %s
*Company:* %s
*GST:* %s

✨ You can now post loads!

Type POST to start posting loads.`,
		createdShipper.ShipperID, createdShipper.CompanyName, createdShipper.GSTNumber), nil
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

	// Format response
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

		return fmt.Sprintf(`📍 *Tracking Details*

*Booking ID:* %s
*Route:* %s → %s
*Status:* %s
*Trucker:* %s
*Amount:* ₹%.0f

Last Update: Just now`,
			booking.BookingID, load.FromCity, load.ToCity,
			booking.Status, booking.TruckerID, booking.AgreedPrice), nil
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

// Handle trucker registration (existing code)
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

// Handle load search (existing code)
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

	// Format response
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

// Handle booking (existing code)
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

	return fmt.Sprintf(`✅ *Booking Confirmed!*

*Booking ID:* %s
*Load ID:* %s
*Route:* %s → %s
*Material:* %s
*Amount:* ₹%.0f
*Your earnings:* ₹%.0f (after 5%% commission)

📱 *OTP for pickup:* %s

Show this OTP at pickup point.

💰 Payment will be credited within 48 hours after delivery!

Type STATUS to check your bookings.`,
		booking.BookingID, load.LoadID, load.FromCity, load.ToCity,
		load.Material, booking.AgreedPrice, booking.NetAmount, booking.OTP), nil
}

// Handle status check (existing code)
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

	// Format response
	response := fmt.Sprintf("📊 *Your Bookings*\n👤 %s (%s)\n\n", trucker.Name, trucker.VehicleNo)

	for i, booking := range bookings {
		if i > 4 { // Limit display
			response += fmt.Sprintf("\n... and %d more bookings", len(bookings)-5)
			break
		}

		// Get load details
		load, _ := w.store.GetLoad(booking.LoadID)
		if load != nil {
			response += fmt.Sprintf(`🚛 *Booking:* %s
📍 *Route:* %s → %s
💰 *Earnings:* ₹%.0f
📊 *Status:* %s

`, booking.BookingID, load.FromCity, load.ToCity,
				booking.NetAmount, booking.Status)
		}
	}

	return response, nil
}
