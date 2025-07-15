package storage

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
)

// MemoryStore holds all data in memory for MVP
type MemoryStore struct {
	mu       sync.RWMutex
	truckers map[uint]*models.Trucker // Changed from string to uint
	loads    map[uint]*models.Load    // Changed from string to uint
	bookings map[uint]*models.Booking // Changed from string to uint
	shippers map[string]*models.Shipper
	otps     map[string]*models.OTP

	// Add these new fields:
	supportTickets map[string]*models.SupportTicket
	verifications  map[string]*models.Verification

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
		shippers:            make(map[string]*models.Shipper),
		otps:                make(map[string]*models.OTP),
		truckersByTruckerID: make(map[string]*models.Trucker),
		loadsByLoadID:       make(map[string]*models.Load),
		bookingsByBookingID: make(map[string]*models.Booking),
		supportTickets:      make(map[string]*models.SupportTicket), // Add this
		verifications:       make(map[string]*models.Verification),  // Add this
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

func (m *MemoryStore) UpdateBooking(booking *models.Booking) error {
	m.bookingMu.Lock()
	defer m.bookingMu.Unlock()

	// Update the booking in both maps
	booking.UpdatedAt = time.Now()

	// Update in the ID map
	if existingBooking, exists := m.bookings[booking.ID]; exists {
		*existingBooking = *booking
	}

	// Update in the BookingID map
	if existingBooking, exists := m.bookingsByBookingID[booking.BookingID]; exists {
		*existingBooking = *booking
	}

	return nil
}

// Shipper operations
func (m *MemoryStore) CreateShipper(shipper *models.Shipper) (*models.Shipper, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if phone already exists
	for _, s := range m.shippers {
		if s.Phone == shipper.Phone {
			return nil, fmt.Errorf("phone number already registered")
		}
		if s.GSTNumber == shipper.GSTNumber {
			return nil, fmt.Errorf("GST number already registered")
		}
	}

	// Generate ShipperID
	shipper.ID = uint(len(m.shippers) + 1)
	shipper.ShipperID = fmt.Sprintf("SH%05d", shipper.ID)
	shipper.CreatedAt = time.Now()
	shipper.UpdatedAt = time.Now()

	m.shippers[shipper.ShipperID] = shipper
	return shipper, nil
}

func (m *MemoryStore) GetShipper(id string) (*models.Shipper, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if shipper, ok := m.shippers[id]; ok {
		return shipper, nil
	}

	// Try numeric ID
	for _, shipper := range m.shippers {
		if fmt.Sprintf("%d", shipper.ID) == id {
			return shipper, nil
		}
	}

	return nil, fmt.Errorf("shipper not found")
}

func (m *MemoryStore) GetShipperByPhone(phone string) (*models.Shipper, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, shipper := range m.shippers {
		if shipper.Phone == phone {
			return shipper, nil
		}
	}
	return nil, fmt.Errorf("shipper not found")
}

func (m *MemoryStore) GetShipperByGST(gst string) (*models.Shipper, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, shipper := range m.shippers {
		if shipper.GSTNumber == gst {
			return shipper, nil
		}
	}
	return nil, fmt.Errorf("shipper not found")
}

func (m *MemoryStore) GetLoadsByShipper(shipperID string) ([]*models.Load, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var loads []*models.Load
	for _, load := range m.loads {
		if load.ShipperID == shipperID {
			loads = append(loads, load)
		}
	}

	// Sort by created date (newest first)
	sort.Slice(loads, func(i, j int) bool {
		return loads[i].CreatedAt.After(loads[j].CreatedAt)
	})

	return loads, nil
}

// OTP operations
func (m *MemoryStore) CreateOTP(otp *models.OTP) (*models.OTP, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate ID
	otp.ID = uint(len(m.otps) + 1)
	otp.CreatedAt = time.Now()
	otp.UpdatedAt = time.Now()

	// Store using phone+code+purpose as key
	key := fmt.Sprintf("%s:%s:%s", otp.Phone, otp.Code, otp.Purpose)
	m.otps[key] = otp

	return otp, nil
}

func (m *MemoryStore) GetActiveOTP(phone, code, purpose string) (*models.OTP, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s:%s", phone, code, purpose)
	otp, exists := m.otps[key]
	if !exists {
		return nil, fmt.Errorf("OTP not found or invalid")
	}

	if otp.IsUsed {
		return nil, fmt.Errorf("OTP already used")
	}

	return otp, nil
}

func (m *MemoryStore) UpdateOTP(otp *models.OTP) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", otp.Phone, otp.Code, otp.Purpose)
	m.otps[key] = otp

	return nil
}

func (m *MemoryStore) GetOTPByReference(referenceID, purpose string) (*models.OTP, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, otp := range m.otps {
		if otp.ReferenceID == referenceID && otp.Purpose == purpose && !otp.IsUsed {
			return otp, nil
		}
	}

	return nil, fmt.Errorf("OTP not found")
}

// GetAllTruckers returns all truckers
func (m *MemoryStore) GetAllTruckers() ([]*models.Trucker, error) {
	m.truckerMu.RLock()
	defer m.truckerMu.RUnlock()

	truckers := make([]*models.Trucker, 0, len(m.truckers))
	for _, trucker := range m.truckers {
		truckers = append(truckers, trucker)
	}
	return truckers, nil
}

// GetAvailableTruckers returns all available truckers
func (m *MemoryStore) GetAvailableTruckers() ([]*models.Trucker, error) {
	m.truckerMu.RLock()
	defer m.truckerMu.RUnlock()

	var truckers []*models.Trucker
	for _, trucker := range m.truckers {
		if trucker.Available && !trucker.IsSuspended {
			truckers = append(truckers, trucker)
		}
	}
	return truckers, nil
}

// UpdateTrucker updates a trucker
func (m *MemoryStore) UpdateTrucker(trucker *models.Trucker) error {
	m.truckerMu.Lock()
	defer m.truckerMu.Unlock()

	trucker.UpdatedAt = time.Now()
	m.truckers[trucker.ID] = trucker
	m.truckersByTruckerID[trucker.TruckerID] = trucker
	return nil
}

// GetTruckerByID returns a trucker by ID (same as GetTrucker)
func (m *MemoryStore) GetTruckerByID(truckerID string) (*models.Trucker, error) {
	return m.GetTrucker(truckerID)
}

// GetShipperByID returns a shipper by ID (same as GetShipper)
func (m *MemoryStore) GetShipperByID(shipperID string) (*models.Shipper, error) {
	return m.GetShipper(shipperID)
}

// UpdateShipper updates a shipper
func (m *MemoryStore) UpdateShipper(shipper *models.Shipper) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	shipper.UpdatedAt = time.Now()
	m.shippers[shipper.ShipperID] = shipper
	return nil
}

// GetAllShippers returns all shippers
func (m *MemoryStore) GetAllShippers() ([]*models.Shipper, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shippers := make([]*models.Shipper, 0, len(m.shippers))
	for _, shipper := range m.shippers {
		shippers = append(shippers, shipper)
	}
	return shippers, nil
}

// UpdateLoad updates a load
func (m *MemoryStore) UpdateLoad(load *models.Load) error {
	m.loadMu.Lock()
	defer m.loadMu.Unlock()

	load.UpdatedAt = time.Now()
	m.loads[load.ID] = load
	m.loadsByLoadID[load.LoadID] = load
	return nil
}

// GetLoadsByStatus returns loads by status
func (m *MemoryStore) GetLoadsByStatus(status string) ([]*models.Load, error) {
	m.loadMu.RLock()
	defer m.loadMu.RUnlock()

	var loads []*models.Load
	for _, load := range m.loads {
		if load.Status == status {
			loads = append(loads, load)
		}
	}
	return loads, nil
}

// GetExpiredLoads returns expired loads
func (m *MemoryStore) GetExpiredLoads() ([]*models.Load, error) {
	return m.GetLoadsByStatus("expired")
}

// GetBookingsByStatus returns bookings by status
func (m *MemoryStore) GetBookingsByStatus(status string) ([]*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	var bookings []*models.Booking
	for _, booking := range m.bookings {
		if booking.Status == status {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

// GetBookingsByPaymentStatus returns bookings by payment status
func (m *MemoryStore) GetBookingsByPaymentStatus(paymentStatus string) ([]*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	var bookings []*models.Booking
	for _, booking := range m.bookings {
		if booking.PaymentStatus == paymentStatus {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

// GetActiveBookings returns all active bookings
func (m *MemoryStore) GetActiveBookings() ([]*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	var bookings []*models.Booking
	activeStatuses := map[string]bool{
		"confirmed":        true,
		"trucker_assigned": true,
		"in_transit":       true,
	}

	for _, booking := range m.bookings {
		if activeStatuses[booking.Status] {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

// GetCompletedBookingsInDateRange returns completed bookings in date range
func (m *MemoryStore) GetCompletedBookingsInDateRange(startDate, endDate string) ([]*models.Booking, error) {
	m.bookingMu.RLock()
	defer m.bookingMu.RUnlock()

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	var bookings []*models.Booking
	for _, booking := range m.bookings {
		if booking.Status == "delivered" &&
			booking.CreatedAt.After(start) &&
			booking.CreatedAt.Before(end.Add(24*time.Hour)) {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

// DeleteExpiredOTPs deletes expired OTPs
func (m *MemoryStore) DeleteExpiredOTPs() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, otp := range m.otps {
		if otp.ExpiresAt.Before(now) {
			delete(m.otps, key)
		}
	}
	return nil
}

// Analytics operations - Note: These are stub implementations for memory store
func (m *MemoryStore) GetTruckerStats(truckerID string) (*models.TruckerStats, error) {
	// In memory store, we calculate stats on the fly
	stats := &models.TruckerStats{
		TruckerID: truckerID,
	}

	// Calculate from bookings
	bookings, _ := m.GetBookingsByTrucker(truckerID)
	for _, b := range bookings {
		if b.Status == "delivered" {
			stats.CompletedTrips++
			stats.TotalEarnings += b.NetAmount
		}
	}

	return stats, nil
}

func (m *MemoryStore) GetShipperStats(shipperID string) (*models.ShipperStats, error) {
	// In memory store, we calculate stats on the fly
	stats := &models.ShipperStats{
		ShipperID: shipperID,
	}

	// Calculate from loads
	loads, _ := m.GetLoadsByShipper(shipperID)
	stats.TotalLoads = len(loads)
	for _, l := range loads {
		if l.Status == "available" || l.Status == "booked" {
			stats.ActiveLoads++
		} else if l.Status == "delivered" || l.Status == "completed" {
			stats.CompletedLoads++
		}
	}

	return stats, nil
}

func (m *MemoryStore) GetTruckersWithExpiringDocuments(daysAhead int) ([]*models.Trucker, error) {
	m.truckerMu.RLock()
	defer m.truckerMu.RUnlock()

	var truckers []*models.Trucker
	expiryDate := time.Now().AddDate(0, 0, daysAhead)

	for _, trucker := range m.truckers {
		if trucker.DocumentExpiryDate != nil &&
			trucker.DocumentExpiryDate.Before(expiryDate) &&
			trucker.DocumentExpiryDate.After(time.Now()) {
			truckers = append(truckers, trucker)
		}
	}
	return truckers, nil
}

func (m *MemoryStore) GetInactiveTruckers(daysSinceLastActive int) ([]*models.Trucker, error) {
	m.truckerMu.RLock()
	defer m.truckerMu.RUnlock()

	var truckers []*models.Trucker
	cutoffDate := time.Now().AddDate(0, 0, -daysSinceLastActive)

	for _, trucker := range m.truckers {
		if trucker.UpdatedAt.Before(cutoffDate) {
			truckers = append(truckers, trucker)
		}
	}
	return truckers, nil
}

func (m *MemoryStore) GetInactiveShippers(daysSinceLastActive int) ([]*models.Shipper, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var shippers []*models.Shipper
	cutoffDate := time.Now().AddDate(0, 0, -daysSinceLastActive)

	for _, shipper := range m.shippers {
		if shipper.UpdatedAt.Before(cutoffDate) {
			shippers = append(shippers, shipper)
		}
	}
	return shippers, nil
}

// Support operations
func (m *MemoryStore) CreateSupportTicket(ticket *models.SupportTicket) (*models.SupportTicket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ticket.ID = uint(len(m.supportTickets) + 1)
	if ticket.TicketID == "" {
		ticket.TicketID = fmt.Sprintf("TK%d", time.Now().Unix())
	}
	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	if m.supportTickets == nil {
		m.supportTickets = make(map[string]*models.SupportTicket)
	}
	m.supportTickets[ticket.TicketID] = ticket
	return ticket, nil
}

func (m *MemoryStore) GetSupportTicket(ticketID string) (*models.SupportTicket, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if ticket, exists := m.supportTickets[ticketID]; exists {
		return ticket, nil
	}
	return nil, fmt.Errorf("ticket not found")
}

func (m *MemoryStore) GetSupportTicketsByUser(userPhone string) ([]*models.SupportTicket, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tickets []*models.SupportTicket
	for _, ticket := range m.supportTickets {
		if ticket.UserPhone == userPhone {
			tickets = append(tickets, ticket)
		}
	}
	return tickets, nil
}

func (m *MemoryStore) UpdateSupportTicket(ticket *models.SupportTicket) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ticket.UpdatedAt = time.Now()
	m.supportTickets[ticket.TicketID] = ticket
	return nil
}

// Admin operations
func (m *MemoryStore) GetPendingVerifications() ([]*models.Verification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var verifications []*models.Verification
	for _, v := range m.verifications {
		if v.Status == "pending" {
			verifications = append(verifications, v)
		}
	}
	return verifications, nil
}

func (m *MemoryStore) UpdateVerificationStatus(verificationID string, status string, adminNotes string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if v, exists := m.verifications[verificationID]; exists {
		v.Status = status
		v.AdminNotes = adminNotes
		now := time.Now()
		v.VerifiedAt = &now
		v.UpdatedAt = now
		return nil
	}
	return fmt.Errorf("verification not found")
}

func (m *MemoryStore) SuspendAccount(userType string, userID string, reason string) error {
	if userType == "trucker" {
		m.truckerMu.Lock()
		defer m.truckerMu.Unlock()

		for _, trucker := range m.truckers {
			if trucker.TruckerID == userID {
				trucker.IsSuspended = true
				trucker.UpdatedAt = time.Now()
				return nil
			}
		}
	} else if userType == "shipper" {
		m.mu.Lock()
		defer m.mu.Unlock()

		if shipper, exists := m.shippers[userID]; exists {
			shipper.Active = false
			shipper.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

func (m *MemoryStore) ReactivateAccount(userType string, userID string) error {
	if userType == "trucker" {
		m.truckerMu.Lock()
		defer m.truckerMu.Unlock()

		for _, trucker := range m.truckers {
			if trucker.TruckerID == userID {
				trucker.IsSuspended = false
				trucker.UpdatedAt = time.Now()
				return nil
			}
		}
	} else if userType == "shipper" {
		m.mu.Lock()
		defer m.mu.Unlock()

		if shipper, exists := m.shippers[userID]; exists {
			shipper.Active = true
			shipper.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

// GetVerification helper method
func (m *MemoryStore) GetVerification(verificationID string) (*models.Verification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if v, exists := m.verifications[verificationID]; exists {
		return v, nil
	}
	return nil, fmt.Errorf("verification not found")
}
