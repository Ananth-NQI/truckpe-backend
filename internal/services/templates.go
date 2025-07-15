package services

import (
	"fmt"
)

// TemplateConfig holds template configuration
type TemplateConfig struct {
	SID         string
	Description string
	Parameters  []string
	ButtonType  string // "quick_reply", "call_to_action", "none"
}

// TemplateService handles WhatsApp template operations
type TemplateService struct {
	twilioService *TwilioService
}

// NewTemplateService creates a new template service
func NewTemplateService(twilioService *TwilioService) *TemplateService {
	return &TemplateService{
		twilioService: twilioService,
	}
}

// WhatsAppTemplates maps template names to their Twilio Content SIDs
var WhatsAppTemplates = map[string]TemplateConfig{
	// Critical Templates (11)
	"trucker_booked_notification": {
		SID:         "HX47e05263be82ccfcf1dc742cfa2ad048",
		Description: "Notification when trucker books a load",
		Parameters:  []string{"trucker_name", "load_id", "route", "amount"},
		ButtonType:  "quick_reply",
	},
	"registration_success": {
		SID:         "HX35ef139e281e6cbe5d40289a65d7a5f6",
		Description: "Successful registration confirmation",
		Parameters:  []string{"name", "user_id", "vehicle_number"},
		ButtonType:  "quick_reply",
	},
	"delivery_notification_shipper": {
		SID:         "HX76b216121ff7a0d87323e6613c908345",
		Description: "Notify shipper about delivery",
		Parameters:  []string{"load_id", "delivery_time", "trucker_name"},
		ButtonType:  "quick_reply",
	},
	"verification_approved": {
		SID:         "HX1e355b7c7317ecf67b530b1352579e2e",
		Description: "Document verification approved",
		Parameters:  []string{"name", "document_type"},
		ButtonType:  "quick_reply",
	},
	"emergency_sos": {
		SID:         "HX7241f24041c5b415ebd2d2b5b914a64d",
		Description: "Emergency assistance",
		Parameters:  []string{"trucker_name", "location", "vehicle_number"},
		ButtonType:  "quick_reply",
	},
	"trucker_delayed": {
		SID:         "HXf1476dc22fee732cdc58de7af342eb32",
		Description: "Trucker delay notification",
		Parameters:  []string{"booking_id", "new_eta", "reason"},
		ButtonType:  "quick_reply",
	},
	"delivery_confirmation": {
		SID:         "HXc181e33f1b0320a1e31d9c0c0d31fadb",
		Description: "Delivery completion confirmation",
		Parameters:  []string{"booking_id", "delivered_at", "amount"},
		ButtonType:  "quick_reply",
	},
	"price_negotiation_request": {
		SID:         "HXd58d99f56d21070706f6e7d966177d39",
		Description: "Price negotiation from trucker",
		Parameters:  []string{"trucker_name", "load_id", "current_price", "requested_price"},
		ButtonType:  "quick_reply",
	},
	"breakdown_assistance": {
		SID:         "HX3b350931197012609eac18182f519069",
		Description: "Vehicle breakdown help",
		Parameters:  []string{"trucker_name", "location", "issue"},
		ButtonType:  "quick_reply",
	},
	"verification_rejected": {
		SID:         "HX74af70fcbf0b6868d08cb982b0434e26",
		Description: "Document verification rejected",
		Parameters:  []string{"name", "document_type", "reason"},
		ButtonType:  "quick_reply",
	},
	"session_expired": {
		SID:         "HX50694296a3c4c48b625930edb62816c6",
		Description: "Session timeout notification",
		Parameters:  []string{},
		ButtonType:  "quick_reply",
	},

	// Important Templates (5)
	"rate_experience": {
		SID:         "HXdb9a092550107764d3030340760d1f60",
		Description: "Rate delivery experience",
		Parameters:  []string{"booking_id", "route"},
		ButtonType:  "quick_reply",
	},
	"verification_pending": {
		SID:         "HXdea5188924298b26a3a54e675d256ffa",
		Description: "Verification in progress",
		Parameters:  []string{"name", "document_type"},
		ButtonType:  "quick_reply",
	},
	"load_expired_notification": {
		SID:         "HX0ee8359336142e588390ce02539b9e8d",
		Description: "Load expired",
		Parameters:  []string{"load_id", "route"},
		ButtonType:  "quick_reply",
	},
	"payment_reminder": {
		SID:         "HX9e16296d3858800848d8c2bfa48f92f5",
		Description: "Payment reminder",
		Parameters:  []string{"amount", "booking_id", "due_date"},
		ButtonType:  "quick_reply",
	},
	"account_suspended": {
		SID:         "HX457cd80a6e456b2090e578ad707dc7ba",
		Description: "Account suspension notice",
		Parameters:  []string{"name", "reason"},
		ButtonType:  "quick_reply",
	},

	// Nice-to-have Templates (10)
	"referral_program": {
		SID:         "HX76e9ac1a9ca21ecd2fcb0a28a0a5d79c",
		Description: "Referral program invitation",
		Parameters:  []string{"name", "referral_code", "bonus_amount"},
		ButtonType:  "quick_reply",
	},
	"weekly_summary": {
		SID:         "HX093f16a3d23b3a5fa1a4cbca4b7ac39a",
		Description: "Weekly earnings summary",
		Parameters:  []string{"name", "trips_count", "earnings", "top_route"},
		ButtonType:  "quick_reply",
	},
	"document_expiry_reminder": {
		SID:         "HX5cbe9e7ed99dc01dbe6a153feab4169d",
		Description: "Document expiry reminder",
		Parameters:  []string{"document_type", "expiry_date"},
		ButtonType:  "quick_reply",
	},
	"milestone_achievement": {
		SID:         "HX4f5625a542697deb13319127a5757827",
		Description: "Milestone celebration",
		Parameters:  []string{"name", "milestone", "reward"},
		ButtonType:  "quick_reply",
	},
	"route_suggestion": {
		SID:         "HX5ff9d491eb8f769b6d26d4e5f27e1f05",
		Description: "Suggested profitable route",
		Parameters:  []string{"route", "avg_price", "demand_level"},
		ButtonType:  "quick_reply",
	},
	"maintenance_reminder": {
		SID:         "HX36605ff993a7613962b745ed650a226d",
		Description: "Vehicle maintenance reminder",
		Parameters:  []string{"vehicle_number", "service_type", "last_service"},
		ButtonType:  "quick_reply",
	},
	"festival_greeting": {
		SID:         "HXaef2c345befe7f9d81f99532b744c2f3",
		Description: "Festival wishes",
		Parameters:  []string{"name", "festival_name"},
		ButtonType:  "quick_reply",
	},
	"inactivity_reminder": {
		SID:         "HX13fa6b8571408522538477740b5fb7b3",
		Description: "Inactivity reminder",
		Parameters:  []string{"name", "days_inactive"},
		ButtonType:  "quick_reply",
	},
	"bulk_load_alert": {
		SID:         "HXa67b6674ab43752c980d8ee1a30d8684",
		Description: "Bulk loads available",
		Parameters:  []string{"route", "load_count", "total_value"},
		ButtonType:  "quick_reply",
	},
	"support_ticket_update": {
		SID:         "HXa66e11c65043ad67c47bb0bef506ed9b",
		Description: "Support ticket status",
		Parameters:  []string{"ticket_id", "status", "message"},
		ButtonType:  "quick_reply",
	},

	// Original Templates (18)
	"payment_processed": {
		SID:         "HXe63676d2ffb658b9ebc8c07ba9339d8d",
		Description: "Payment processed",
		Parameters:  []string{"amount", "booking_id", "transaction_id"},
		ButtonType:  "quick_reply",
	},
	"booking_cancelled": {
		SID:         "HX844446a76eed6e4404926312feac533a",
		Description: "Booking cancellation",
		Parameters:  []string{"booking_id", "reason"},
		ButtonType:  "quick_reply",
	},
	"welcome_message": {
		SID:         "HX44dec15f87f391428a523a9c4bddd83d",
		Description: "Welcome message",
		Parameters:  []string{},
		ButtonType:  "quick_reply",
	},
	"trucker_arrived_notify": {
		SID:         "HXe6c3cbc1e7a696f064edca198de05f10",
		Description: "Trucker arrival notification",
		Parameters:  []string{"trucker_name", "vehicle_number", "booking_id"},
		ButtonType:  "quick_reply",
	},
	"load_match_notification": {
		SID:         "HX96787d7ad76432fa238d6101b733fdd6",
		Description: "Matching load found",
		Parameters:  []string{"route", "price", "load_id"},
		ButtonType:  "quick_reply",
	},
	"pickup_completed": {
		SID:         "HX3edf55dceb9d1cd7179c98d634413224",
		Description: "Pickup completion",
		Parameters:  []string{"booking_id", "pickup_time"},
		ButtonType:  "quick_reply",
	},
	"shipper_otp_share_v2": {
		SID:         "HX836dc551fded3fb3038daf655144a363",
		Description: "OTP for shipper v2",
		Parameters:  []string{"otp", "trucker_name", "booking_id"},
		ButtonType:  "quick_reply",
	},
	"welcome_trucker": {
		SID:         "HX810a291483b94d1bef97384e90d75d06",
		Description: "Welcome trucker",
		Parameters:  []string{"name"},
		ButtonType:  "quick_reply",
	},
	"delivery_complete": {
		SID:         "HX67dd32397fc0a0028c3d3bb3dd61076b",
		Description: "Delivery completed",
		Parameters:  []string{"booking_id", "delivered_at"},
		ButtonType:  "quick_reply",
	},
	"load_posted_confirm": {
		SID:         "HX4154cdd87874ed19216457f3bce90109",
		Description: "Load posted confirmation",
		Parameters:  []string{"load_id", "route", "price"},
		ButtonType:  "quick_reply",
	},
	"post_load_easy": {
		SID:         "HX63d8632210d16e18475177a125e7d766",
		Description: "Easy load posting",
		Parameters:  []string{},
		ButtonType:  "quick_reply",
	},
	"booking_actions_v2": {
		SID:         "HX5712caba664f67a1b3442899a7c3c075",
		Description: "Booking actions v2",
		Parameters:  []string{"booking_id"},
		ButtonType:  "quick_reply",
	},
	"load_selection": {
		SID:         "HXab48e69265176ec946b52cba912f4820",
		Description: "Load selection list",
		Parameters:  []string{},
		ButtonType:  "list_picker",
	},
	"booking_actions": {
		SID:         "HX7c55ff32adbd560e1e88b15716c4bd28",
		Description: "Booking actions",
		Parameters:  []string{"booking_id"},
		ButtonType:  "quick_reply",
	},

	// Latest Flow Templates
	"new_user_welcome": {
		SID:         "HX9e3c1f89b63fcca20366bae6b929ec87",
		Description: "Welcome message for new users with role selection",
		Parameters:  []string{},
		ButtonType:  "quick_reply",
	},
	"trucker_registration_name": {
		SID:         "HX788b577b3ec0e48f0d3e677db8d14d5d",
		Description: "Ask for trucker's name during registration",
		Parameters:  []string{},
		ButtonType:  "none",
	},
	"vehicle_type_selection": {
		SID:         "HX20b60e5931450edd071cde08ba5e7774",
		Description: "Vehicle type selection buttons",
		Parameters:  []string{},
		ButtonType:  "quick_reply",
	},
	"vehicle_type_selection_more": {
		SID:         "HXcd56d35b153c46249a86999c48133f23",
		Description: "Additional vehicle type options",
		Parameters:  []string{},
		ButtonType:  "quick_reply",
	},
	"registration_confirmation": {
		SID:         "HX6b4934d86a771f2d08053cdf34a69f69",
		Description: "Confirm registration details",
		Parameters:  []string{"name", "vehicle_number", "vehicle_type", "capacity"},
		ButtonType:  "quick_reply",
	},
	"trucker_main_menu": {
		SID:         "HXf3e91c01396d3a0dd8a3df21626f350d",
		Description: "Main menu for registered truckers",
		Parameters:  []string{"greeting", "name"},
		ButtonType:  "quick_reply",
	},
}

// SendTemplate sends a WhatsApp template with parameters
func (ts *TemplateService) SendTemplate(to string, templateName string, params map[string]string) error {
	template, exists := WhatsAppTemplates[templateName]
	if !exists {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	// Validate required parameters
	for _, requiredParam := range template.Parameters {
		if _, ok := params[requiredParam]; !ok {
			return fmt.Errorf("missing required parameter: %s", requiredParam)
		}
	}

	// Convert parameters to format Twilio expects
	contentVariables := make(map[string]string)
	for i, paramName := range template.Parameters {
		if value, ok := params[paramName]; ok {
			// Twilio uses {{1}}, {{2}}, etc.
			contentVariables[fmt.Sprintf("%d", i+1)] = value
		}
	}

	// Send via Twilio
	return ts.twilioService.SendWhatsAppTemplate(to, template.SID, contentVariables)
}

// GetTemplateInfo returns information about a template
func (ts *TemplateService) GetTemplateInfo(templateName string) (*TemplateConfig, error) {
	template, exists := WhatsAppTemplates[templateName]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", templateName)
	}
	return &template, nil
}
