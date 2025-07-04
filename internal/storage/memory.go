package storage

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
)

// MemoryStore holds all data in memory for MVP
type MemoryStore struct {
	truckers map[uint]*models.Trucker // Changed from string to uint
	loads    map[uint]*models.Load    // Changed from string to uint
	bookings map[uint]*models.Booking // Changed from string to uint

	// Maps for lookup by string IDs
	truckersByTruckerID map[string]*models.Trucker
	loadsByLoadID       map[string]*models.Load
	bookingsByBookingID map[string]*models.Booking

	// Mutexes for thread safety
	truckerMu sync.RWMutex
	loadMu    sync.RWMutex
	bookingMu sync.RWMutex

	// Counters for ID generation
	truckerCounter uint
	loadCounter    uint
	bookingCounter uint
}

// NewMemoryStore creates a new in-memory storage
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		truckers:            make(map[uint]*models.Trucker),
		loads:               make(map[uint]*models.Load),
		bookings:            make(map[uint]*models.Booking),
		truckersByTruckerID: make(map[string]*models.Trucker),
		loadsByLoadID:       make(map[string]*models.Load),
		bookingsByBookingID: make(map[string]*models.Booking),
	}
}

// Trucker operations
func (m *MemoryStore) CreateTrucker(reg *models.TruckerRegistration) (*models.Trucker, error) {
	m.truckerMu.Lock()
	defer m.truckerMu.Unlock()

	// Check if phone already exists
	for _, t := range m.truckers {
		if t.Phone == reg.Phone {
			return nil, fmt.Errorf("phone number already registered")
		}
		if t.VehicleNo == reg.VehicleNo {
			return nil, fmt.Errorf("vehicle already registered")
		}
	}

	m.truckerCounter++
	now := time.Now()

	trucker := &models.Trucker{
		TruckerID:   fmt.Sprintf("TRK%05d", m.truckerCounter),
		Name:        reg.Name,
		Phone:       reg.Phone,
		VehicleNo:   reg.VehicleNo,
		VehicleType: reg.VehicleType,
		Capacity:    reg.Capacity,
		Verified:    false,
		Rating:      5.0,
		TotalTrips:  0,
		Available:   true,
	}

	// Set ID and timestamps (simulating GORM behavior)
	trucker.ID = m.truckerCounter
	trucker.CreatedAt = now
	trucker.UpdatedAt = now

	m.truckers[trucker.ID] = trucker
	m.truckersByTruckerID[trucker.TruckerID] = trucker

	return trucker, nil
}

func (m *MemoryStore) GetTrucker(id string) (*models.Trucker, error) {
	m.truckerMu.RLock()
	defer m.truckerMu.RUnlock()

	// Try to find by TruckerID first
	if trucker, exists := m.truckersByTruckerID[id]; exists {
		return trucker, nil
	}

	// Try to parse as uint ID
	var uintID uint
	if _, err := fmt.Sscanf(id, "%d", &uintID); err == nil {
		if trucker, exists := m.truckers[uintID]; exists {
			return trucker, nil
		}
	}

	return nil, fmt.Errorf("trucker not found")
}

func (m *MemoryStore) GetTruckerByPhone(phone string) (*models.Trucker, error) {
	m.truckerMu.RLock()
	defer m.truckerMu.RUnlock()

	for _, trucker := range m.truckers {
		if trucker.Phone == phone {
			return trucker, nil
		}
	}
	return nil, fmt.Errorf("trucker not found")
}

// Load operations
func (m *MemoryStore) CreateLoad(load *models.Load) (*models.Load, error) {
	m.loadMu.Lock()
	defer m.loadMu.Unlock()

	m.loadCounter++
	now := time.Now()

	load.ID = m.loadCounter
	load.LoadID = fmt.Sprintf("LD%05d", m.loadCounter)
	load.Status = "available"
	load.CreatedAt = now
	load.UpdatedAt = now

	m.loads[load.ID] = load
	m.loadsByLoadID[load.LoadID] = load

	return load, nil
}

func (m *MemoryStore) GetLoad(id string) (*models.Load, error) {
	m.loadMu.RLock()
	defer m.loadMu.RUnlock()

	// Try to find by LoadID first
	if load, exists := m.loadsByLoadID[id]; exists {
		return load, nil
	}

	// Try to parse as uint ID
	var uintID uint
	if _, err := fmt.Sscanf(id, "%d", &uintID); err == nil {
		if load, exists := m.loads[uintID]; exists {
			return load, nil
		}
	}

	return nil, fmt.Errorf("load not found")
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

		// Case-insensitive city matching
		if search.FromCity != "" && !strings.EqualFold(load.FromCity, search.FromCity) {
			continue
		}
		if search.ToCity != "" && !strings.EqualFold(load.ToCity, search.ToCity) {
			continue
		}
		if search.VehicleType != "" && !strings.Contains(strings.ToLower(load.VehicleType), strings.ToLower(search.VehicleType)) {
			continue
		}
		if search.DateFrom != "" {
			if date, err := time.Parse("2006-01-02", search.DateFrom); err == nil {
				if load.LoadingDate.Before(date) {
					continue
				}
			}
		}

		results = append(results, load)
	}
	return results, nil
}

func (m *MemoryStore) UpdateLoadStatus(id string, status string) error {
	m.loadMu.Lock()
	defer m.loadMu.Unlock()

	// Try LoadID first
	if load, exists := m.loadsByLoadID[id]; exists {
		load.Status = status
		load.UpdatedAt = time.Now()
		return nil
	}

	// Try uint ID
	var uintID uint
	if _, err := fmt.Sscanf(id, "%d", &uintID); err == nil {
		if load, exists := m.loads[uintID]; exists {
			load.Status = status
			load.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("load not found")
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
	trucker, err := m.GetTrucker(truckerID)
	if err != nil {
		return nil, err
	}
	if !trucker.Available {
		return nil, fmt.Errorf("trucker not available")
	}

	m.bookingMu.Lock()
	defer m.bookingMu.Unlock()

	m.bookingCounter++
	now := time.Now()

	booking := &models.Booking{
		BookingID:     fmt.Sprintf("BK%05d", m.bookingCounter),
		LoadID:        loadID,
		TruckerID:     truckerID,
		ShipperID:     load.ShipperID,
		AgreedPrice:   load.Price,
		Commission:    load.Price * 0.05, // 5% commission
		NetAmount:     load.Price * 0.95,
		Status:        models.BookingStatusConfirmed,
		PaymentStatus: models.PaymentStatusPending,
		ConfirmedAt:   &now,
		OTP:           fmt.Sprintf("%06d", time.Now().Unix()%1000000), // Generate 6-digit OTP
	}

	// Set ID and timestamps
	booking.ID = m.bookingCounter
	booking.CreatedAt = now
	booking.UpdatedAt = now

	// Update load status
	m.loadMu.Lock()
	load.Status = "booked"
	load.UpdatedAt = now
	m.loadMu.Unlock()

	// Update trucker availability
	m.truckerMu.Lock()
	trucker.Available = false
	trucker.UpdatedAt = now
	m.truckerMu.Unlock()

	m.bookings[booking.ID] = booking
	m.bookingsByBookingID[booking.BookingID] = booking

	return booking, nil
}

func (m *MemoryStore) GetBooking(id string) (*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	// Try to find by BookingID first
	if booking, exists := m.bookingsByBookingID[id]; exists {
		return booking, nil
	}

	// Try to parse as uint ID
	var uintID uint
	if _, err := fmt.Sscanf(id, "%d", &uintID); err == nil {
		if booking, exists := m.bookings[uintID]; exists {
			return booking, nil
		}
	}

	return nil, fmt.Errorf("booking not found")
}

func (m *MemoryStore) GetBookingsByTrucker(truckerID string) ([]*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	var bookings []*models.Booking
	for _, booking := range m.bookings {
		if booking.TruckerID == truckerID {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

func (m *MemoryStore) GetBookingsByLoad(loadID string) ([]*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	var bookings []*models.Booking
	for _, booking := range m.bookings {
		if booking.LoadID == loadID {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

func (m *MemoryStore) UpdateBookingStatus(id string, status string) error {
	m.bookingMu.Lock()
	defer m.bookingMu.Unlock()

	var booking *models.Booking

	// Try BookingID first
	if b, exists := m.bookingsByBookingID[id]; exists {
		booking = b
	} else {
		// Try uint ID
		var uintID uint
		if _, err := fmt.Sscanf(id, "%d", &uintID); err == nil {
			if b, exists := m.bookings[uintID]; exists {
				booking = b
			}
		}
	}

	if booking == nil {
		return fmt.Errorf("booking not found")
	}

	booking.Status = status
	booking.UpdatedAt = time.Now()

	// Update timestamps based on status
	now := time.Now()
	switch status {
	case models.BookingStatusInTransit:
		booking.PickedUpAt = &now
	case models.BookingStatusDelivered:
		booking.DeliveredAt = &now
		// Also mark load as delivered
		m.loadMu.Lock()
		if load, err := m.GetLoad(booking.LoadID); err == nil {
			load.Status = "delivered"
			load.UpdatedAt = now
		}
		m.loadMu.Unlock()
		// Mark trucker as available again
		m.truckerMu.Lock()
		if trucker, err := m.GetTrucker(booking.TruckerID); err == nil {
			trucker.Available = true
			trucker.TotalTrips++
			trucker.UpdatedAt = now
		}
		m.truckerMu.Unlock()
	case models.BookingStatusCompleted:
		booking.CompletedAt = &now
		booking.PaymentStatus = models.PaymentStatusCompleted
	}

	return nil
}
