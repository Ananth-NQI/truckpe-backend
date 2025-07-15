package storage

import (
	"sync"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
)

var (
	storeInstance Store
	storeOnce     sync.Once
)

// SetStore sets the global store instance (call from main.go)
func SetStore(s Store) {
	storeInstance = s
}

// GetStore returns the global store instance
func GetStore() Store {
	return storeInstance
}

// Store defines the interface for storage operations
type Store interface {
	// Trucker operations
	CreateTrucker(reg *models.TruckerRegistration) (*models.Trucker, error)
	GetTrucker(id string) (*models.Trucker, error)
	GetTruckerByID(truckerID string) (*models.Trucker, error)
	GetTruckerByPhone(phone string) (*models.Trucker, error)
	GetAllTruckers() ([]*models.Trucker, error)
	GetAvailableTruckers() ([]*models.Trucker, error)
	UpdateTrucker(trucker *models.Trucker) error

	// Load operations
	CreateLoad(load *models.Load) (*models.Load, error)
	GetLoad(id string) (*models.Load, error)
	GetAvailableLoads() ([]*models.Load, error)
	SearchLoads(search *models.LoadSearch) ([]*models.Load, error)
	UpdateLoadStatus(id string, status string) error
	UpdateLoad(load *models.Load) error
	GetLoadsByStatus(status string) ([]*models.Load, error)
	GetExpiredLoads() ([]*models.Load, error)

	// Booking operations
	CreateBooking(loadID, truckerID string) (*models.Booking, error)
	GetBooking(id string) (*models.Booking, error)
	GetBookingsByTrucker(truckerID string) ([]*models.Booking, error)
	GetBookingsByLoad(loadID string) ([]*models.Booking, error)
	GetBookingsByStatus(status string) ([]*models.Booking, error)
	GetBookingsByPaymentStatus(paymentStatus string) ([]*models.Booking, error)
	UpdateBookingStatus(id string, status string) error
	UpdateBooking(booking *models.Booking) error
	GetActiveBookings() ([]*models.Booking, error)
	GetCompletedBookingsInDateRange(startDate, endDate string) ([]*models.Booking, error)

	// Shipper operations
	CreateShipper(shipper *models.Shipper) (*models.Shipper, error)
	GetShipper(id string) (*models.Shipper, error)
	GetShipperByID(shipperID string) (*models.Shipper, error)
	GetShipperByPhone(phone string) (*models.Shipper, error)
	GetShipperByGST(gst string) (*models.Shipper, error)
	GetLoadsByShipper(shipperID string) ([]*models.Load, error)
	UpdateShipper(shipper *models.Shipper) error
	GetAllShippers() ([]*models.Shipper, error)

	// OTP operations
	CreateOTP(otp *models.OTP) (*models.OTP, error)
	GetActiveOTP(phone, code, purpose string) (*models.OTP, error)
	UpdateOTP(otp *models.OTP) error
	GetOTPByReference(referenceID, purpose string) (*models.OTP, error)
	DeleteExpiredOTPs() error

	// Analytics operations (for scheduled jobs)
	GetTruckerStats(truckerID string) (*models.TruckerStats, error)
	GetShipperStats(shipperID string) (*models.ShipperStats, error)
	GetTruckersWithExpiringDocuments(daysAhead int) ([]*models.Trucker, error)
	GetInactiveTruckers(daysSinceLastActive int) ([]*models.Trucker, error)
	GetInactiveShippers(daysSinceLastActive int) ([]*models.Shipper, error)

	// Support operations
	CreateSupportTicket(ticket *models.SupportTicket) (*models.SupportTicket, error)
	GetSupportTicket(ticketID string) (*models.SupportTicket, error)
	GetSupportTicketsByUser(userPhone string) ([]*models.SupportTicket, error)
	UpdateSupportTicket(ticket *models.SupportTicket) error

	// Admin operations
	GetPendingVerifications() ([]*models.Verification, error)
	UpdateVerificationStatus(verificationID string, status string, adminNotes string) error
	SuspendAccount(userType string, userID string, reason string) error
	ReactivateAccount(userType string, userID string) error
	GetVerification(verificationID string) (*models.Verification, error)
}
