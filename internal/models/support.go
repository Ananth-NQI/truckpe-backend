package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// In internal/models/support.go, update the SupportTicket struct:

type SupportTicket struct {
	gorm.Model
	TicketID    string     `gorm:"uniqueIndex;not null" json:"ticket_id"`
	UserPhone   string     `gorm:"index;not null" json:"user_phone"`
	UserType    string     `json:"user_type"`  // trucker or shipper
	UserID      string     `json:"user_id"`    // TruckerID or ShipperID
	IssueType   string     `json:"issue_type"` // ADD THIS LINE - payment, booking, technical, general
	Description string     `json:"description"`
	Status      string     `gorm:"default:'open'" json:"status"`     // open, in_progress, resolved, closed
	Priority    string     `gorm:"default:'medium'" json:"priority"` // low, medium, high, urgent
	AssignedTo  string     `json:"assigned_to,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Resolution  string     `json:"resolution,omitempty"`
}

// Also add these constants for issue types (if they don't exist):
const (
	IssueTypePayment   = "payment"
	IssueTypeBooking   = "booking"
	IssueTypeTechnical = "technical"
	IssueTypeGeneral   = "general"
	IssueTypeComplaint = "complaint"
)

// In the BeforeCreate hook, add default issue type:
func (st *SupportTicket) BeforeCreate(tx *gorm.DB) error {
	if st.TicketID == "" {
		st.TicketID = fmt.Sprintf("TK%d", time.Now().UnixNano())
	}

	// Set default issue type if not provided
	if st.IssueType == "" {
		st.IssueType = IssueTypeGeneral
	}

	return nil
}
