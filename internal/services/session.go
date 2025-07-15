package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// Session represents a user session
type Session struct {
	SessionID  string                 `json:"session_id"`
	UserPhone  string                 `json:"user_phone"`
	UserType   string                 `json:"user_type"` // "trucker" or "shipper"
	UserID     string                 `json:"user_id"`
	UserName   string                 `json:"user_name"`
	CreatedAt  time.Time              `json:"created_at"`
	LastActive time.Time              `json:"last_active"`
	ExpiresAt  time.Time              `json:"expires_at"`
	IsActive   bool                   `json:"is_active"`
	Context    map[string]interface{} `json:"context"` // For storing conversation context
}

// SessionManager manages user sessions
type SessionManager struct {
	store         storage.Store
	twilioService *TwilioService
	sessions      map[string]*Session // In-memory session storage
	mu            sync.RWMutex
	sessionTTL    time.Duration
}

// Singleton instance
var (
	sessionManagerInstance *SessionManager
	sessionManagerOnce     sync.Once
)

// NewSessionManager creates a new session manager
func NewSessionManager(store storage.Store, twilioService *TwilioService) *SessionManager {
	sm := &SessionManager{
		store:         store,
		twilioService: twilioService,
		sessions:      make(map[string]*Session),
		sessionTTL:    30 * time.Minute, // 30 minute session timeout
	}

	// Start cleanup routine
	go sm.cleanupExpiredSessions()

	return sm
}

// GetSessionManager returns the singleton session manager instance
func GetSessionManager() *SessionManager {
	sessionManagerOnce.Do(func() {
		// This will be initialized once when first called
		if sessionManagerInstance == nil {
			log.Println("Warning: SessionManager not initialized. Creating new instance.")
			// This is a temporary solution - you should initialize this properly in main.go
			sessionManagerInstance = &SessionManager{
				sessions:   make(map[string]*Session),
				sessionTTL: 30 * time.Minute,
			}
		}
	})
	return sessionManagerInstance
}

// SetSessionManager sets the global session manager instance (call from main.go)
func SetSessionManager(sm *SessionManager) {
	sessionManagerInstance = sm
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(userPhone, userType, userID, userName string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if session already exists
	if existingSession, exists := sm.sessions[userPhone]; exists && existingSession.IsActive {
		// Update last active time
		existingSession.LastActive = time.Now()
		existingSession.ExpiresAt = time.Now().Add(sm.sessionTTL)
		return existingSession, nil
	}

	// Create new session
	session := &Session{
		SessionID:  fmt.Sprintf("SES%d", time.Now().UnixNano()),
		UserPhone:  userPhone,
		UserType:   userType,
		UserID:     userID,
		UserName:   userName,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		ExpiresAt:  time.Now().Add(sm.sessionTTL),
		IsActive:   true,
		Context:    make(map[string]interface{}),
	}

	sm.sessions[userPhone] = session
	log.Printf("Session created for %s (%s)", userName, userPhone)

	return session, nil
}

// GetSession retrieves an active session
func (sm *SessionManager) GetSession(userPhone string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// UpdateSessionActivity updates the last active time of a session
func (sm *SessionManager) UpdateSessionActivity(userPhone string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.LastActive = time.Now()
	session.ExpiresAt = time.Now().Add(sm.sessionTTL)

	return nil
}

// UpdateSessionContext updates the session context (for multi-step flows)
func (sm *SessionManager) UpdateSessionContext(userPhone string, key string, value interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Context[key] = value
	session.LastActive = time.Now()
	session.ExpiresAt = time.Now().Add(sm.sessionTTL)

	return nil
}

// GetSessionContext retrieves a value from session context
func (sm *SessionManager) GetSessionContext(userPhone string, key string) (interface{}, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	value, exists := session.Context[key]
	if !exists {
		return nil, fmt.Errorf("key not found in context")
	}

	return value, nil
}

// ExpireSession manually expires a session
func (sm *SessionManager) ExpireSession(userPhone string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.IsActive = false
	session.ExpiresAt = time.Now()

	// Send session expired notification
	sm.sendSessionExpiredNotification(session)

	// Remove from active sessions
	delete(sm.sessions, userPhone)

	log.Printf("Session expired for %s (%s)", session.UserName, userPhone)
	return nil
}

// sendSessionExpiredNotification sends the session expired template
func (sm *SessionManager) sendSessionExpiredNotification(session *Session) {
	if sm.twilioService == nil {
		log.Printf("Cannot send session expired notification - twilioService is nil")
		return
	}

	templateService := NewTemplateService(sm.twilioService)

	// Calculate session duration
	duration := session.LastActive.Sub(session.CreatedAt)
	durationMinutes := int(duration.Minutes())

	params := map[string]string{
		"name":             session.UserName,
		"session_duration": fmt.Sprintf("%d minutes", durationMinutes),
		"last_activity":    session.LastActive.Format("3:04 PM"),
	}

	err := templateService.SendTemplate(session.UserPhone, "session_expired", params)
	if err != nil {
		log.Printf("Failed to send session expired template to %s: %v", session.UserPhone, err)
	}
}

// cleanupExpiredSessions runs periodically to clean up expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()

		expiredSessions := []*Session{}

		// Find expired sessions
		for phone, session := range sm.sessions {
			if time.Now().After(session.ExpiresAt) && session.IsActive {
				expiredSessions = append(expiredSessions, session)
				session.IsActive = false
				delete(sm.sessions, phone)
			}
		}

		sm.mu.Unlock()

		// Send notifications for expired sessions
		for _, session := range expiredSessions {
			sm.sendSessionExpiredNotification(session)
			log.Printf("Cleaned up expired session for %s", session.UserPhone)
		}
	}
}

// GetActiveSessions returns all active sessions (for monitoring)
func (sm *SessionManager) GetActiveSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	activeSessions := []*Session{}
	for _, session := range sm.sessions {
		if session.IsActive && time.Now().Before(session.ExpiresAt) {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

// ExtendSession extends the session timeout
func (sm *SessionManager) ExtendSession(userPhone string, additionalMinutes int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.ExpiresAt = session.ExpiresAt.Add(time.Duration(additionalMinutes) * time.Minute)
	log.Printf("Session extended for %s by %d minutes", session.UserName, additionalMinutes)

	return nil
}

// SessionStats provides session statistics
type SessionStats struct {
	ActiveSessions   int            `json:"active_sessions"`
	TotalSessions    int            `json:"total_sessions"`
	SessionsByType   map[string]int `json:"sessions_by_type"`
	AverageeDuration float64        `json:"average_duration_minutes"`
}

// GetSessionStats returns current session statistics
func (sm *SessionManager) GetSessionStats() *SessionStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := &SessionStats{
		ActiveSessions: 0,
		TotalSessions:  len(sm.sessions),
		SessionsByType: make(map[string]int),
	}

	totalDuration := 0.0
	activeCount := 0

	for _, session := range sm.sessions {
		if session.IsActive && time.Now().Before(session.ExpiresAt) {
			stats.ActiveSessions++
			activeCount++

			// Count by type
			stats.SessionsByType[session.UserType]++

			// Calculate duration
			duration := time.Now().Sub(session.CreatedAt).Minutes()
			totalDuration += duration
		}
	}

	if activeCount > 0 {
		stats.AverageeDuration = totalDuration / float64(activeCount)
	}

	return stats
}

// Multi-step flow support for complex interactions

// StartMultiStepFlow initiates a multi-step interaction
func (sm *SessionManager) StartMultiStepFlow(userPhone, flowType string, initialData map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Set flow context
	session.Context["flow_type"] = flowType
	session.Context["flow_step"] = 1
	session.Context["flow_data"] = initialData
	session.Context["flow_started_at"] = time.Now()

	log.Printf("Started %s flow for %s", flowType, session.UserName)
	return nil
}

// GetCurrentFlow retrieves the current flow information
func (sm *SessionManager) GetCurrentFlow(userPhone string) (flowType string, step int, data map[string]interface{}, err error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return "", 0, nil, fmt.Errorf("session not found")
	}

	flowType, _ = session.Context["flow_type"].(string)
	step, _ = session.Context["flow_step"].(int)
	data, _ = session.Context["flow_data"].(map[string]interface{})

	return flowType, step, data, nil
}

// AdvanceFlow moves to the next step in a multi-step flow
func (sm *SessionManager) AdvanceFlow(userPhone string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return fmt.Errorf("session not found")
	}

	currentStep, _ := session.Context["flow_step"].(int)
	session.Context["flow_step"] = currentStep + 1

	return nil
}

// CompleteFlow completes a multi-step flow
func (sm *SessionManager) CompleteFlow(userPhone string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[userPhone]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Clear flow context
	delete(session.Context, "flow_type")
	delete(session.Context, "flow_step")
	delete(session.Context, "flow_data")
	delete(session.Context, "flow_started_at")

	log.Printf("Completed flow for %s", session.UserName)
	return nil
}
