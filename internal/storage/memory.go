package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
)

// MemoryStore holds all data in memory for MVP
type MemoryStore struct {
	truckers map[string]*models.Trucker
	loads    map[string]*models.Load
	bookings map[string]*models.Booking

	// Mutexes for thread safety
	truckerMu sync.RWMutex
	loadMu    sync.RWMutex
	bookingMu sync.RWMutex

	// Counters for ID generation
	truckerCounter int
	loadCounter    int
	bookingCounter int
}

// NewMemoryStore creates a new in-memory storage
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		truckers: make(map[string]*models.Trucker),
		loads:    make(map[string]*models.Load),
		bookings: make(map[string]*models.Booking),
	}
}

// Trucker operations
func (m *MemoryStore) CreateTrucker(reg *models.TruckerRegistration) (*models.Trucker, error) {
	m.truckerMu.Lock()
	defer m.truckerMu.Unlock()

	m.truckerCounter++
	trucker := &models.Trucker{
		ID:          fmt.Sprintf("TRK%05d", m.truckerCounter),
		Name:        reg.Name,
		Phone:       reg.Phone,
		VehicleNo:   reg.VehicleNo,
		VehicleType: reg.VehicleType,
		Capacity:    reg.Capacity,
		Verified:    false,
		Rating:      0.0,
		TotalTrips:  0,
		Available:   true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.truckers[trucker.ID] = trucker
	return trucker, nil
}

func (m *MemoryStore) GetTrucker(id string) (*models.Trucker, error) {
	m.truckerMu.RLock()
	defer m.truckerMu.RUnlock()

	trucker, exists := m.truckers[id]
	if !exists {
		return nil, fmt.Errorf("trucker not found")
	}
	return trucker, nil
}

// Load operations
func (m *MemoryStore) CreateLoad(load *models.Load) (*models.Load, error) {
	m.loadMu.Lock()
	defer m.loadMu.Unlock()

	m.loadCounter++
	load.ID = fmt.Sprintf("LD%05d", m.loadCounter)
	load.Status = "available"
	load.CreatedAt = time.Now()
	load.UpdatedAt = time.Now()

	m.loads[load.ID] = load
	return load, nil
}

func (m *MemoryStore) GetLoad(id string) (*models.Load, error) {
	m.loadMu.RLock()
	defer m.loadMu.RUnlock()

	load, exists := m.loads[id]
	if !exists {
		return nil, fmt.Errorf("load not found")
	}
	return load, nil
}

func (m *MemoryStore) GetAvailableLoads() ([]*models.Load, error) {
	m.loadMu.RLock()
	defer m.loadMu.RUnlock()

	var loads []*models.Load
	for _, load := range m.loads {
		if load.Status == "available" {
			loads = append(loads, load)
		}
	}
	return loads, nil
}

func (m *MemoryStore) SearchLoads(search *models.LoadSearch) ([]*models.Load, error) {
	m.loadMu.RLock()
	defer m.loadMu.RUnlock()

	var results []*models.Load
	for _, load := range m.loads {
		if load.Status != "available" {
			continue
		}

		// Match criteria
		if search.FromCity != "" && load.FromCity != search.FromCity {
			continue
		}
		if search.ToCity != "" && load.ToCity != search.ToCity {
			continue
		}
		if search.VehicleType != "" && load.VehicleType != search.VehicleType {
			continue
		}

		results = append(results, load)
	}
	return results, nil
}

// Booking operations
func (m *MemoryStore) CreateBooking(loadID, truckerID string) (*models.Booking, error) {
	// First check if load exists and is available
	load, err := m.GetLoad(loadID)
	if err != nil {
		return nil, err
	}
	if load.Status != "available" {
		return nil, fmt.Errorf("load not available")
	}

	// Check if trucker exists
	_, err = m.GetTrucker(truckerID)
	if err != nil {
		return nil, err
	}

	m.bookingMu.Lock()
	defer m.bookingMu.Unlock()

	m.bookingCounter++
	now := time.Now()

	booking := &models.Booking{
		ID:            fmt.Sprintf("BK%05d", m.bookingCounter),
		LoadID:        loadID,
		TruckerID:     truckerID,
		ShipperID:     load.ShipperID,
		AgreedPrice:   load.Price,
		Commission:    load.Price * 0.05, // 5% commission
		NetAmount:     load.Price * 0.95,
		Status:        models.BookingStatusConfirmed,
		PaymentStatus: models.PaymentStatusPending,
		ConfirmedAt:   &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Update load status
	m.loadMu.Lock()
	load.Status = "booked"
	load.UpdatedAt = now
	m.loadMu.Unlock()

	m.bookings[booking.ID] = booking
	return booking, nil
}

func (m *MemoryStore) GetBooking(id string) (*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	booking, exists := m.bookings[id]
	if !exists {
		return nil, fmt.Errorf("booking not found")
	}
	return booking, nil
}
