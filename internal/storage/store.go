package storage

import "github.com/Ananth-NQI/truckpe-backend/internal/models"

// Store defines the interface for storage operations
type Store interface {
	// Trucker operations
	CreateTrucker(reg *models.TruckerRegistration) (*models.Trucker, error)
	GetTrucker(id string) (*models.Trucker, error)
	GetTruckerByPhone(phone string) (*models.Trucker, error)

	// Load operations
	CreateLoad(load *models.Load) (*models.Load, error)
	GetLoad(id string) (*models.Load, error)
	GetAvailableLoads() ([]*models.Load, error)
	SearchLoads(search *models.LoadSearch) ([]*models.Load, error)
	UpdateLoadStatus(id string, status string) error

	// Booking operations
	CreateBooking(loadID, truckerID string) (*models.Booking, error)
	GetBooking(id string) (*models.Booking, error)
	GetBookingsByTrucker(truckerID string) ([]*models.Booking, error)
	GetBookingsByLoad(loadID string) ([]*models.Booking, error)
	UpdateBookingStatus(id string, status string) error

	// SHIPPER OPERATIONS:
	CreateShipper(shipper *models.Shipper) (*models.Shipper, error)
	GetShipper(id string) (*models.Shipper, error)
	GetShipperByPhone(phone string) (*models.Shipper, error)
	GetShipperByGST(gst string) (*models.Shipper, error)
	GetLoadsByShipper(shipperID string) ([]*models.Load, error)
}
