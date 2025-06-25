package services

import (
	"fmt"
	"strings"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// WhatsAppService handles WhatsApp message processing
type WhatsAppService struct {
	store *storage.MemoryStore
}

// NewWhatsAppService creates a new WhatsApp service
func NewWhatsAppService(store *storage.MemoryStore) *WhatsAppService {
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
		trucker.ID, trucker.Name, trucker.VehicleNo,
		trucker.VehicleType, trucker.Capacity), nil
}

// Handle load search
func (w *WhatsAppService) handleLoadSearch(phone, msg string) (string, error) {
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
	response := fmt.Sprintf("🚛 *Available Loads from %s*\n\n", search.FromCity)

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

`, load.ID, load.FromCity, load.ToCity, load.Material,
			load.Weight, load.Price, load.VehicleType)
	}

	response += "To book, type: BOOK <Load_ID>\nExample: BOOK LD00001"
	return response, nil
}

// Handle booking
func (w *WhatsAppService) handleBooking(phone, msg string) (string, error) {
	// Extract load ID
	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return "❌ Please specify Load ID\n\nExample: BOOK LD00001", nil
	}

	loadID := parts[1]

	// TODO: Find trucker by phone number
	// For now, return instruction to register first
	return fmt.Sprintf(`📦 Booking Load: %s

⚠️ Please make sure you're registered first!

If not registered, type:
REGISTER Name, VehicleNo, Type, Capacity

If registered, booking feature coming soon!`, loadID), nil
}

// Handle status check
func (w *WhatsAppService) handleStatus(phone string) (string, error) {
	return "📊 *Your Status*\n\nNo active bookings.\n\nSearch for loads: LOAD <from> <to>", nil
}
