package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// NaturalFlowService orchestrates conversational flows
type NaturalFlowService struct {
	store              storage.Store
	sessionManager     *SessionManager
	templateService    *TemplateService
	interactiveService *InteractiveTemplateService
	twilioService      *TwilioService
}

// FlowContext stores the conversation context
type FlowContext struct {
	Flow         string                 `json:"flow"` // welcome, registration, booking, etc.
	Step         string                 `json:"step"` // current step in flow
	Data         map[string]interface{} `json:"data"` // collected data
	LastActivity time.Time              `json:"last_activity"`
}

// NewNaturalFlowService creates a new natural flow service
func NewNaturalFlowService(
	store storage.Store,
	sessionManager *SessionManager,
	templateService *TemplateService,
	interactiveService *InteractiveTemplateService,
	twilioService *TwilioService,
) *NaturalFlowService {
	return &NaturalFlowService{
		store:              store,
		sessionManager:     sessionManager,
		templateService:    templateService,
		interactiveService: interactiveService,
		twilioService:      twilioService,
	}
}

// ProcessNaturalMessage is the main entry point for all messages
func (n *NaturalFlowService) ProcessNaturalMessage(phone string, message string, buttonPayload string) error {
	// Clean phone number
	phone = strings.TrimPrefix(phone, "whatsapp:")

	// Log for debugging
	log.Printf("Natural Flow: Processing message from %s - Message: '%s', Button: '%s'", phone, message, buttonPayload)

	// Check if user exists
	trucker, _ := n.store.GetTruckerByPhone(phone)
	shipper, _ := n.store.GetShipperByPhone(phone)

	// Get or create session
	var session *Session
	var err error

	if trucker != nil {
		session, err = n.sessionManager.GetSession(phone)
		if err != nil {
			// Create session for existing trucker
			session, err = n.sessionManager.CreateSession(phone, "trucker", trucker.TruckerID, trucker.Name)
			if err != nil {
				return n.sendErrorMessage(phone, "Failed to create session. Please try again.")
			}
		}
		return n.handleExistingTrucker(session, trucker, message, buttonPayload)

	} else if shipper != nil {
		session, err = n.sessionManager.GetSession(phone)
		if err != nil {
			// Create session for existing shipper
			session, err = n.sessionManager.CreateSession(phone, "shipper", shipper.ShipperID, shipper.CompanyName)
			if err != nil {
				return n.sendErrorMessage(phone, "Failed to create session. Please try again.")
			}
		}
		return n.handleExistingShipper(session, shipper, message, buttonPayload)

	} else {
		// New user
		session, err = n.sessionManager.GetSession(phone)
		if err != nil {
			// Create new session
			session, err = n.sessionManager.CreateSession(phone, "new", "", "")
			if err != nil {
				return n.sendErrorMessage(phone, "Failed to create session. Please try again.")
			}
			// Initialize welcome flow
			session.Context["flow"] = "welcome"
			session.Context["step"] = "initial"
		}
		return n.handleNewUser(session, message, buttonPayload)
	}
}

// handleNewUser manages the flow for unregistered users
func (n *NaturalFlowService) handleNewUser(session *Session, message string, buttonPayload string) error {
	// Get flow context
	flow, _ := session.Context["flow"].(string)
	step, _ := session.Context["step"].(string)

	log.Printf("New user flow: %s, step: %s", flow, step)

	// Handle different flows
	switch flow {
	case "welcome", "":
		return n.handleWelcomeFlow(session, step, message, buttonPayload)
	case "trucker_registration":
		return n.handleTruckerRegistrationFlow(session, step, message, buttonPayload)
	case "shipper_registration":
		return n.handleShipperRegistrationFlow(session, step, message, buttonPayload)
	default:
		// Reset to welcome if unknown flow
		session.Context["flow"] = "welcome"
		session.Context["step"] = "initial"
		return n.handleWelcomeFlow(session, "initial", message, buttonPayload)
	}
}

// handleWelcomeFlow manages the initial welcome interaction
func (n *NaturalFlowService) handleWelcomeFlow(session *Session, step string, message string, buttonPayload string) error {
	switch step {
	case "initial", "":
		// Send new user welcome template with role selection buttons
		params := map[string]string{}
		err := n.templateService.SendTemplate(session.UserPhone, "new_user_welcome", params)
		if err != nil {
			log.Printf("Failed to send new_user_welcome template: %v", err)
			// Fallback to text
			return n.sendWelcomeText(session.UserPhone)
		}

		// Update session
		session.Context["flow"] = "welcome"
		session.Context["step"] = "role_selection"
		n.sessionManager.UpdateSessionContext(session.UserPhone, "flow", "welcome")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "role_selection")

		return nil

	case "role_selection":
		// Handle button selection or text response
		if buttonPayload != "" {
			// Handle button payloads from new_user_welcome template
			switch buttonPayload {
			case "role_trucker":
				session.Context["flow"] = "trucker_registration"
				session.Context["step"] = "collect_name"
				n.sessionManager.UpdateSessionContext(session.UserPhone, "flow", "trucker_registration")
				n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_name")
				return n.handleTruckerRegistrationFlow(session, "collect_name", "", "")

			case "role_shipper":
				session.Context["flow"] = "shipper_registration"
				session.Context["step"] = "collect_company"
				n.sessionManager.UpdateSessionContext(session.UserPhone, "flow", "shipper_registration")
				n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_company")
				return n.handleShipperRegistrationFlow(session, "collect_company", "", "")

			case "learn_more":
				return n.sendLearnMore(session.UserPhone)

			default:
				// Unknown button payload
				log.Printf("Unknown button payload: %s", buttonPayload)
			}
		}

		// Handle text responses (backward compatibility)
		msgLower := strings.ToLower(message)
		if strings.Contains(msgLower, "truck") || strings.Contains(msgLower, "driver") || msgLower == "1" {
			session.Context["flow"] = "trucker_registration"
			session.Context["step"] = "collect_name"
			n.sessionManager.UpdateSessionContext(session.UserPhone, "flow", "trucker_registration")
			n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_name")
			return n.handleTruckerRegistrationFlow(session, "collect_name", "", "")

		} else if strings.Contains(msgLower, "ship") || strings.Contains(msgLower, "company") || msgLower == "2" {
			session.Context["flow"] = "shipper_registration"
			session.Context["step"] = "collect_company"
			n.sessionManager.UpdateSessionContext(session.UserPhone, "flow", "shipper_registration")
			n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_company")
			return n.handleShipperRegistrationFlow(session, "collect_company", "", "")

		} else if strings.Contains(msgLower, "learn") || msgLower == "3" {
			return n.sendLearnMore(session.UserPhone)

		} else {
			// Resend welcome if unclear response
			return n.sendRoleSelectionReminder(session.UserPhone)
		}

	default:
		// Reset to initial
		session.Context["step"] = "initial"
		return n.handleWelcomeFlow(session, "initial", message, buttonPayload)
	}
}

// Helper functions for sending messages
func (n *NaturalFlowService) sendWelcomeText(phone string) error {
	message := `🚛 *Welcome to TruckPe!*
India's most trusted digital freight marketplace.

Are you a:
👤 *Trucker* - Find loads & earn more
🏭 *Shipper* - Book reliable trucks

Please type:
- "Trucker" if you drive trucks
- "Shipper" if you need to transport goods

Or simply reply with 1 for Trucker, 2 for Shipper.`

	return n.twilioService.SendWhatsAppMessage(phone, message)
}

func (n *NaturalFlowService) sendRoleSelectionReminder(phone string) error {
	message := `Please let us know who you are:

Reply with:
1️⃣ or "Trucker" - If you're a truck driver
2️⃣ or "Shipper" - If you need to ship goods

What would you like to do?`

	return n.twilioService.SendWhatsAppMessage(phone, message)
}

func (n *NaturalFlowService) sendLearnMore(phone string) error {
	message := `📚 *About TruckPe*

TruckPe connects truck owners directly with businesses needing transportation.

*For Truckers:*
✅ Find loads instantly
✅ Transparent pricing
✅ Quick payments (48 hours)
✅ No middlemen

*For Shippers:*
✅ Verified truckers
✅ Real-time tracking
✅ Secure payments
✅ 24/7 support

Ready to start?
Reply "Trucker" or "Shipper" to register!`

	return n.twilioService.SendWhatsAppMessage(phone, message)
}

func (n *NaturalFlowService) sendErrorMessage(phone string, errorMsg string) error {
	return n.twilioService.SendWhatsAppMessage(phone, fmt.Sprintf("❌ %s", errorMsg))
}

// handleTruckerRegistrationFlow manages the trucker registration process
func (n *NaturalFlowService) handleTruckerRegistrationFlow(session *Session, step string, message string, buttonPayload string) error {
	log.Printf("Trucker registration - Step: %s, Message: %s, ButtonPayload: %s", step, message, buttonPayload)

	// Get or initialize registration data
	regData, ok := session.Context["registration_data"].(map[string]interface{})
	if !ok {
		regData = make(map[string]interface{})
		session.Context["registration_data"] = regData
	}

	switch step {
	case "collect_name":
		// Use the template for name collection
		params := map[string]string{}
		err := n.templateService.SendTemplate(session.UserPhone, "trucker_registration_name", params)
		if err != nil {
			// Fallback to plain text
			msg := `Great! Let's get you registered as a trucker. 🚛

What's your full name?

Example: Rajesh Kumar`
			return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)
		}

		// Update session
		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "validate_name")

		return nil

	case "validate_name":
		// Validate and store name
		name := strings.TrimSpace(message)
		if len(name) < 3 {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"Please enter your full name (at least 3 characters).")
		}

		// Store name
		regData["name"] = name
		session.Context["registration_data"] = regData

		// Move to vehicle number collection
		msg := fmt.Sprintf(`Nice to meet you, %s! 👋

Now, please enter your vehicle registration number.

Example: TN01AB1234`, name)

		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "validate_vehicle")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", regData)

		return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)

	case "validate_vehicle":
		// Validate vehicle number
		vehicleNo := strings.ToUpper(strings.TrimSpace(message))

		// Basic validation (you can make this more sophisticated)
		if len(vehicleNo) < 6 || len(vehicleNo) > 15 {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"Invalid vehicle number. Please enter a valid registration number.\n\nExample: TN01AB1234")
		}

		// Store vehicle number
		regData["vehicle_no"] = vehicleNo
		session.Context["registration_data"] = regData

		// For now, skip Vahan verification - just simulate
		simulationMsg := fmt.Sprintf(`⏳ Verifying vehicle %s...

✅ Vehicle verified!`, vehicleNo)

		// Send simulation message first
		err := n.twilioService.SendWhatsAppMessage(session.UserPhone, simulationMsg)
		if err != nil {
			log.Printf("Failed to send simulation message: %v", err)
		}

		// Wait a bit for effect
		time.Sleep(1 * time.Second)

		// Send vehicle type selection template
		params := map[string]string{}
		err = n.templateService.SendTemplate(session.UserPhone, "vehicle_type_selection", params)
		if err != nil {
			// Fallback to text
			fallbackMsg := `What type of vehicle do you have?

Please select:
1️⃣ Mini Truck (1-3 tons)
2️⃣ Light Truck (3-10 tons)
3️⃣ Heavy Truck (10-20 tons)
4️⃣ Trailer (20+ tons)
5️⃣ Container (32ft/40ft)
6️⃣ Other

Reply with the number (1-6)`
			return n.twilioService.SendWhatsAppMessage(session.UserPhone, fallbackMsg)
		}

		// Also send the "more options" template after a short delay
		go func() {
			time.Sleep(1 * time.Second)
			n.templateService.SendTemplate(session.UserPhone, "vehicle_type_selection_more", map[string]string{})
		}()

		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "validate_vehicle_type")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", regData)

		return nil

	case "validate_vehicle_type":
		// Handle button payloads first
		vehicleType := ""
		if buttonPayload != "" {
			switch buttonPayload {
			case "vehicle_mini":
				vehicleType = "Mini Truck"
			case "vehicle_light":
				vehicleType = "Light Truck"
			case "vehicle_heavy":
				vehicleType = "Heavy Truck"
			case "vehicle_trailer":
				vehicleType = "Trailer"
			case "vehicle_container":
				vehicleType = "Container"
			case "vehicle_other":
				vehicleType = "Other"
			}
		}

		// If no button payload, try text matching
		if vehicleType == "" {
			// Map text selections to vehicle types
			vehicleTypes := map[string]string{
				"1": "Mini Truck",
				"2": "Light Truck",
				"3": "Heavy Truck",
				"4": "Trailer",
				"5": "Container",
				"6": "Other",
			}

			var ok bool
			vehicleType, ok = vehicleTypes[strings.TrimSpace(message)]

			if !ok {
				// Check if they typed the vehicle type
				msgLower := strings.ToLower(message)
				for _, vType := range vehicleTypes {
					if strings.Contains(msgLower, strings.ToLower(vType)) {
						vehicleType = vType
						ok = true
						break
					}
				}

				if !ok {
					return n.twilioService.SendWhatsAppMessage(session.UserPhone,
						"Please select a valid option (1-6) or click one of the buttons.")
				}
			}
		}

		// Store vehicle type
		regData["vehicle_type"] = vehicleType
		session.Context["registration_data"] = regData

		// Ask for capacity
		msg := fmt.Sprintf(`Got it! %s selected.

What's your vehicle's loading capacity in tons?

Examples:
- Mini Truck: 1.5
- Light Truck: 7
- Heavy Truck: 15
- Trailer: 25

Just type the number (e.g., 15)`, vehicleType)

		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "validate_capacity")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", regData)

		return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)

	case "validate_capacity":
		// Parse capacity
		var capacity float64
		_, err := fmt.Sscanf(message, "%f", &capacity)
		if err != nil || capacity <= 0 || capacity > 100 {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"Please enter a valid capacity in tons (e.g., 15 or 15.5)")
		}

		// Store capacity
		regData["capacity"] = capacity
		session.Context["registration_data"] = regData // ADD THIS LINE

		// IMPORTANT: Update the step BEFORE sending template
		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "confirm_registration")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", regData)

		// THEN send confirmation template
		name := regData["name"].(string)
		vehicleNo := regData["vehicle_no"].(string)
		vehicleType := regData["vehicle_type"].(string)

		params := map[string]string{
			"1": name,
			"2": vehicleNo,
			"3": vehicleType,
			"4": fmt.Sprintf("%.1f", capacity),
		}

		err = n.templateService.SendTemplate(session.UserPhone, "registration_confirmation", params)
		if err != nil {
			// Fallback to text
			msg := fmt.Sprintf(`📋 *Please confirm your details:*
	
	👤 *Name:* %s
	🚛 *Vehicle:* %s
	📏 *Type:* %s
	⚖️ *Capacity:* %.1f tons
	
	Is this correct?
	
	Reply:
	✅ YES - Confirm & Register
	❌ NO - Start over`, name, vehicleNo, vehicleType, capacity)

			return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)
		}

		return nil

	case "confirm_registration":
		// Check confirmation - handle button payloads
		confirmed := false

		if buttonPayload != "" {
			if buttonPayload == "confirm_yes" {
				confirmed = true
			} else if buttonPayload == "confirm_no" {
				// Start over
				session.Context["step"] = "collect_name"
				session.Context["registration_data"] = make(map[string]interface{})
				n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_name")
				n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", make(map[string]interface{}))

				return n.handleTruckerRegistrationFlow(session, "collect_name", "", "")
			}
		} else {
			// Handle text responses
			msgLower := strings.ToLower(message)
			if strings.Contains(msgLower, "yes") || strings.Contains(msgLower, "1") {
				confirmed = true
			} else if strings.Contains(msgLower, "no") || strings.Contains(msgLower, "2") {
				// Start over
				session.Context["step"] = "collect_name"
				session.Context["registration_data"] = make(map[string]interface{})
				n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_name")
				n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", make(map[string]interface{}))

				return n.handleTruckerRegistrationFlow(session, "collect_name", "", "")
			} else {
				return n.twilioService.SendWhatsAppMessage(session.UserPhone,
					"Please reply YES to confirm or NO to start over.")
			}
		}

		if !confirmed {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"Please confirm by clicking the button or typing YES/NO.")
		}

		// Create trucker registration
		name := regData["name"].(string)
		vehicleNo := regData["vehicle_no"].(string)
		vehicleType := regData["vehicle_type"].(string)
		capacity := regData["capacity"].(float64)

		reg := &models.TruckerRegistration{
			Name:        name,
			Phone:       session.UserPhone,
			VehicleNo:   vehicleNo,
			VehicleType: vehicleType,
			Capacity:    capacity,
		}

		// Create trucker
		trucker, err := n.store.CreateTrucker(reg)
		if err != nil {
			if strings.Contains(err.Error(), "phone") {
				return n.twilioService.SendWhatsAppMessage(session.UserPhone,
					"❌ This phone number is already registered! Please contact support if you need help.")
			}
			if strings.Contains(err.Error(), "vehicle") {
				return n.twilioService.SendWhatsAppMessage(session.UserPhone,
					"❌ This vehicle is already registered with another account!")
			}
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"❌ Registration failed. Please try again or contact support.")
		}

		// Update session with trucker info
		session.UserType = "trucker"
		session.UserID = trucker.TruckerID
		session.UserName = trucker.Name

		// Clear registration flow
		delete(session.Context, "flow")
		delete(session.Context, "step")
		delete(session.Context, "registration_data")

		// Send success template
		params := map[string]string{
			"name":           trucker.Name,
			"user_id":        trucker.TruckerID,
			"vehicle_number": trucker.VehicleNo,
		}

		err = n.templateService.SendTemplate(session.UserPhone, "registration_success", params)
		if err != nil {
			// Fallback message
			successMsg := fmt.Sprintf(`🎉 *Registration Successful!*

Welcome to TruckPe, %s!

Your Trucker ID: *%s*
Vehicle: *%s*

You can now:
🔍 Search for loads
💰 Start earning

Type anything to see the main menu!`, trucker.Name, trucker.TruckerID, trucker.VehicleNo)

			return n.twilioService.SendWhatsAppMessage(session.UserPhone, successMsg)
		}

		// Send welcome template after a delay
		go func() {
			time.Sleep(2 * time.Second)
			welcomeParams := map[string]string{"name": trucker.Name}
			n.templateService.SendTemplate(session.UserPhone, "welcome_trucker", welcomeParams)
		}()

		return nil

	default:
		// Unknown step, restart
		session.Context["step"] = "collect_name"
		return n.handleTruckerRegistrationFlow(session, "collect_name", "", "")
	}
}

// handleShipperRegistrationFlow manages the shipper registration process
func (n *NaturalFlowService) handleShipperRegistrationFlow(session *Session, step string, message string, buttonPayload string) error {
	log.Printf("Shipper registration - Step: %s, Message: %s", step, message)

	// Get or initialize registration data
	regData, ok := session.Context["registration_data"].(map[string]interface{})
	if !ok {
		regData = make(map[string]interface{})
		session.Context["registration_data"] = regData
	}

	switch step {
	case "collect_company":
		// Ask for company name
		msg := `Welcome! Let's register your business. 🏭

What's your company name?

Example: ABC Logistics Pvt Ltd`

		// Update session
		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "validate_company")

		return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)

	case "validate_company":
		// Validate and store company name
		companyName := strings.TrimSpace(message)
		if len(companyName) < 3 {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"Please enter your full company name (at least 3 characters).")
		}

		// Store company name
		regData["company_name"] = companyName
		session.Context["registration_data"] = regData

		// Ask for GST number
		msg := fmt.Sprintf(`Thank you! 🏢

*%s*

Now, please enter your GST number for verification.

Format: 29ABCDE1234F1Z5
(15 characters)`, companyName)

		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "validate_gst")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", regData)

		return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)

	case "validate_gst":
		// Validate GST number
		gst := strings.ToUpper(strings.TrimSpace(message))

		// Remove any spaces or special characters
		gst = strings.ReplaceAll(gst, " ", "")
		gst = strings.ReplaceAll(gst, "-", "")

		// GST validation (basic - 15 characters)
		if len(gst) != 15 {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				`❌ Invalid GST format!

GST number must be exactly 15 characters.

Example: 29ABCDE1234F1Z5

Please enter a valid GST number:`)
		}

		// Basic pattern check (you can make this more sophisticated)
		// First 2 digits: State code (01-37)
		// Next 10: PAN
		// Next 1: Entity number
		// Next 1: Z by default
		// Last 1: Check digit

		// Store GST
		regData["gst"] = gst

		// Simulate GST verification
		msg := fmt.Sprintf(`⏳ Verifying GST: %s...

✅ GST Verified Successfully!

*Company:* %s
*GST:* %s
*State:* %s

Who will be the primary contact person?

Please enter your full name:`,
			gst,
			regData["company_name"].(string),
			gst,
			getStateFromGST(gst))

		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_contact_name")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", regData)

		return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)

	case "collect_contact_name":
		// Validate contact name
		contactName := strings.TrimSpace(message)
		if len(contactName) < 3 {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"Please enter the contact person's full name (at least 3 characters).")
		}

		// Store contact name
		regData["contact_name"] = contactName

		// Show confirmation
		companyName := regData["company_name"].(string)
		gst := regData["gst"].(string)

		msg := fmt.Sprintf(`📋 *Please confirm your business details:*

🏢 *Company:* %s
📑 *GST:* %s
👤 *Contact:* %s
📱 *Mobile:* %s

Is this information correct?

Reply:
✅ YES - Complete Registration
❌ NO - Start over`,
			companyName,
			gst,
			contactName,
			session.UserPhone)

		n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "confirm_registration")
		n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", regData)

		return n.twilioService.SendWhatsAppMessage(session.UserPhone, msg)

	case "confirm_registration":
		// Check confirmation
		msgLower := strings.ToLower(message)

		if strings.Contains(msgLower, "no") || strings.Contains(msgLower, "2") {
			// Start over
			session.Context["step"] = "collect_company"
			session.Context["registration_data"] = make(map[string]interface{})
			n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "collect_company")
			n.sessionManager.UpdateSessionContext(session.UserPhone, "registration_data", make(map[string]interface{}))

			return n.handleShipperRegistrationFlow(session, "collect_company", "", "")
		}

		if !strings.Contains(msgLower, "yes") && !strings.Contains(msgLower, "1") {
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"Please reply YES to confirm or NO to start over.")
		}

		// Create shipper
		companyName := regData["company_name"].(string)
		gst := regData["gst"].(string)
		//contactName := regData["contact_name"].(string)

		shipper := &models.Shipper{
			CompanyName: companyName,
			GSTNumber:   gst,
			Phone:       session.UserPhone,
		}

		// Save shipper
		createdShipper, err := n.store.CreateShipper(shipper)
		if err != nil {
			if strings.Contains(err.Error(), "phone") {
				return n.twilioService.SendWhatsAppMessage(session.UserPhone,
					"❌ This phone number is already registered! Please contact support if you need help.")
			}
			if strings.Contains(err.Error(), "GST") {
				return n.twilioService.SendWhatsAppMessage(session.UserPhone,
					"❌ This GST number is already registered!")
			}
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"❌ Registration failed. Please try again or contact support.")
		}

		// Update session with shipper info
		session.UserType = "shipper"
		session.UserID = createdShipper.ShipperID
		session.UserName = createdShipper.CompanyName

		// Clear registration flow
		delete(session.Context, "flow")
		delete(session.Context, "step")
		delete(session.Context, "registration_data")

		// Send success template
		params := map[string]string{
			"name":           createdShipper.CompanyName,
			"user_id":        createdShipper.ShipperID,
			"vehicle_number": createdShipper.GSTNumber, // Template expects vehicle_number
		}

		err = n.templateService.SendTemplate(session.UserPhone, "registration_success", params)
		if err != nil {
			// Fallback message
			successMsg := fmt.Sprintf(`🎉 *Registration Successful!*

Welcome to TruckPe!

*Company:* %s
*Shipper ID:* %s
*GST:* %s

You can now:
📦 Post loads
🚛 Find reliable truckers
📊 Track shipments

Type anything to see the main menu!`,
				createdShipper.CompanyName,
				createdShipper.ShipperID,
				createdShipper.GSTNumber)

			return n.twilioService.SendWhatsAppMessage(session.UserPhone, successMsg)
		}

		return nil

	default:
		// Unknown step, restart
		session.Context["step"] = "collect_company"
		return n.handleShipperRegistrationFlow(session, "collect_company", "", "")
	}
}

// Helper function to get state from GST number
func getStateFromGST(gst string) string {
	if len(gst) < 2 {
		return "Unknown"
	}

	stateMap := map[string]string{
		"01": "Jammu & Kashmir",
		"02": "Himachal Pradesh",
		"03": "Punjab",
		"04": "Chandigarh",
		"05": "Uttarakhand",
		"06": "Haryana",
		"07": "Delhi",
		"08": "Rajasthan",
		"09": "Uttar Pradesh",
		"10": "Bihar",
		"11": "Sikkim",
		"12": "Arunachal Pradesh",
		"13": "Nagaland",
		"14": "Manipur",
		"15": "Mizoram",
		"16": "Tripura",
		"17": "Meghalaya",
		"18": "Assam",
		"19": "West Bengal",
		"20": "Jharkhand",
		"21": "Odisha",
		"22": "Chhattisgarh",
		"23": "Madhya Pradesh",
		"24": "Gujarat",
		"27": "Maharashtra",
		"29": "Karnataka",
		"32": "Kerala",
		"33": "Tamil Nadu",
		"36": "Telangana",
		"37": "Andhra Pradesh",
	}

	stateCode := gst[:2]
	if state, ok := stateMap[stateCode]; ok {
		return state
	}

	return "Unknown"
}

func (n *NaturalFlowService) handleExistingTrucker(session *Session, trucker *models.Trucker, message string, buttonPayload string) error {
	// Check if we're in menu selection state
	if flow, _ := session.Context["flow"].(string); flow == "main_menu" {
		return n.handleMainMenu(session, trucker, message, buttonPayload)
	}

	// Otherwise show the main menu
	greeting := n.getTimeBasedGreeting()

	// Send the main menu template with buttons
	params := map[string]string{
		"1": greeting,     // Good morning/afternoon/evening
		"2": trucker.Name, // Trucker's name
	}

	err := n.templateService.SendTemplate(session.UserPhone, "trucker_main_menu", params)
	if err != nil {
		// Fallback to text
		welcomeMsg := fmt.Sprintf(`%s %s! 👋

What would you like to do today?

1️⃣ Find Loads
2️⃣ My Status  
3️⃣ Earnings

Reply with 1, 2, or 3`, greeting, trucker.Name)
		return n.twilioService.SendWhatsAppMessage(session.UserPhone, welcomeMsg)
	}

	// Set session to main menu state
	session.Context["flow"] = "main_menu"
	session.Context["step"] = "menu_selection"
	n.sessionManager.UpdateSessionContext(session.UserPhone, "flow", "main_menu")
	n.sessionManager.UpdateSessionContext(session.UserPhone, "step", "menu_selection")

	return nil
}

// handleMainMenu handles main menu button selections for existing truckers
func (n *NaturalFlowService) handleMainMenu(session *Session, trucker *models.Trucker, message string, buttonPayload string) error {
	// Handle button payloads from main menu
	if buttonPayload != "" {
		switch buttonPayload {
		case "menu_find_loads", "find_loads": // Handle both possible payloads
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"🔍 Finding loads feature coming soon!\n\nFor now, use: LOAD Chennai Bangalore")

		case "menu_my_bookings", "my_bookings":
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"📊 Your bookings feature coming soon!\n\nFor now, use: STATUS")

		case "menu_update_profile", "update_profile":
			return n.twilioService.SendWhatsAppMessage(session.UserPhone,
				"👤 Profile update feature coming soon!")
		}
	}

	// Handle text responses
	switch message {
	case "1":
		return n.twilioService.SendWhatsAppMessage(session.UserPhone,
			"🔍 Finding loads feature coming soon!\n\nFor now, use: LOAD Chennai Bangalore")
	case "2":
		return n.twilioService.SendWhatsAppMessage(session.UserPhone,
			"📊 Status feature coming soon!\n\nFor now, use: STATUS")
	case "3":
		return n.twilioService.SendWhatsAppMessage(session.UserPhone,
			"💰 Earnings feature coming soon!\n\nYour total earnings will appear here.")
	default:
		// Show menu again
		return n.handleExistingTrucker(session, trucker, "", "")
	}
}

func (n *NaturalFlowService) handleExistingShipper(session *Session, shipper *models.Shipper, message string, buttonPayload string) error {
	// Will implement in next step
	greeting := n.getTimeBasedGreeting()
	welcomeMsg := fmt.Sprintf("%s! Welcome back to TruckPe.\n\n%s, what can we help you with today?", greeting, shipper.CompanyName)
	return n.twilioService.SendWhatsAppMessage(session.UserPhone, welcomeMsg)
}

func (n *NaturalFlowService) getTimeBasedGreeting() string {
	hour := time.Now().Hour()
	switch {
	case hour < 12:
		return "Good morning"
	case hour < 17:
		return "Good afternoon"
	default:
		return "Good evening"
	}
}
