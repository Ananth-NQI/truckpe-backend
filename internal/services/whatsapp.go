package services

import (
	"fmt"
	"strings"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// WhatsAppService handles WhatsApp message processing
type WhatsAppService struct {
	store storage.Store // Changed from *storage.MemoryStore to interface
}

// NewWhatsAppService creates a new WhatsApp service
func NewWhatsAppService(store storage.Store) *WhatsAppService { // Changed parameter type
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

	case strings.HasPrefix(msg, "REGISTER"):
		return w.handleRegistration(phone, msg)

	case strings.HasPrefix(msg, "LOAD"):
		return w.handleLoadSearch(phone, msg)

	case strings.HasPrefix(msg, "BOOK"):
		return w.handleBooking(phone, msg)

	case msg == "STATUS":
		return w.handleStatus(phone)

	default:
		return "❌ Invalid command. Type HELP to see available commands.", nil
	}
}

// Help message
func (w *WhatsAppService) getHelpMessage() string {
	return `🚛 *Welcome to TruckPe!*

Available commands:

📝 *REGISTER* - Register as a trucker
Example: REGISTER Rajesh Kumar, TN01AB1234, 32ft, 25

🔍 *LOAD <from> <to>* - Search available loads
Example: LOAD Delhi Mumbai

📦 *BOOK <load_id>* - Book a load
Example: BOOK LD00001

📊 *STATUS* - Check your current bookings

💰 *Instant payment in 48 hours!*
🔒 *Your money is 100% safe with escrow*

Type any command to start!`
}

// Handle registration
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
