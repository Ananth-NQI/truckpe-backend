package services

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/Ananth-NQI/truckpe-backend/internal/models"
	"github.com/Ananth-NQI/truckpe-backend/internal/storage"
)

// RouteSuggestionService provides intelligent route recommendations
type RouteSuggestionService struct {
	store         storage.Store
	twilioService *TwilioService
}

// NewRouteSuggestionService creates a new route suggestion service
func NewRouteSuggestionService(store storage.Store, twilioService *TwilioService) *RouteSuggestionService {
	return &RouteSuggestionService{
		store:         store,
		twilioService: twilioService,
	}
}

// RouteAnalytics contains route performance data
type RouteAnalytics struct {
	Route           string  `json:"route"`
	FromCity        string  `json:"from_city"`
	ToCity          string  `json:"to_city"`
	AveragePrice    float64 `json:"average_price"`
	LoadFrequency   int     `json:"load_frequency"`
	CompletionRate  float64 `json:"completion_rate"`
	AverageDuration float64 `json:"average_duration_hours"`
	Profitability   float64 `json:"profitability_score"`
}

// TruckerPreferences contains trucker's route preferences
type TruckerPreferences struct {
	PreferredRoutes   []string `json:"preferred_routes"`
	MaxDistance       float64  `json:"max_distance_km"`
	MinPrice          float64  `json:"min_price"`
	PreferredMaterial []string `json:"preferred_material"`
	AvoidCities       []string `json:"avoid_cities"`
}

// AnalyzeRoutes analyzes historical route data
func (r *RouteSuggestionService) AnalyzeRoutes() ([]RouteAnalytics, error) {
	// Get all completed bookings for analysis
	bookings, err := r.store.GetBookingsByStatus(models.BookingStatusDelivered)
	if err != nil {
		return nil, err
	}

	// Route statistics map
	routeStats := make(map[string]*RouteAnalytics)

	for _, booking := range bookings {
		load, err := r.store.GetLoad(booking.LoadID)
		if err != nil {
			continue
		}

		route := fmt.Sprintf("%s-%s", load.FromCity, load.ToCity)

		if _, exists := routeStats[route]; !exists {
			routeStats[route] = &RouteAnalytics{
				Route:    route,
				FromCity: load.FromCity,
				ToCity:   load.ToCity,
			}
		}

		stats := routeStats[route]
		stats.LoadFrequency++
		stats.AveragePrice = (stats.AveragePrice*float64(stats.LoadFrequency-1) + booking.NetAmount) / float64(stats.LoadFrequency)

		// Calculate duration if pickup and delivery times are available
		if booking.PickedUpAt != nil && booking.DeliveredAt != nil {
			duration := booking.DeliveredAt.Sub(*booking.PickedUpAt).Hours()
			stats.AverageDuration = (stats.AverageDuration*float64(stats.LoadFrequency-1) + duration) / float64(stats.LoadFrequency)
		}
	}

	// Convert map to slice and calculate profitability
	var analytics []RouteAnalytics
	for _, stats := range routeStats {
		// Simple profitability score based on price and frequency
		stats.Profitability = (stats.AveragePrice * float64(stats.LoadFrequency)) / 1000
		analytics = append(analytics, *stats)
	}

	// Sort by profitability
	sort.Slice(analytics, func(i, j int) bool {
		return analytics[i].Profitability > analytics[j].Profitability
	})

	return analytics, nil
}

// GetTruckerRecommendations gets personalized route recommendations for a trucker
func (r *RouteSuggestionService) GetTruckerRecommendations(truckerID string) ([]RouteAnalytics, error) {
	// Get trucker details
	_, err := r.store.GetTruckerByID(truckerID)
	if err != nil {
		return nil, err
	}

	// Get trucker's booking history
	bookings, err := r.store.GetBookingsByTrucker(truckerID)
	if err != nil {
		return nil, err
	}

	// Analyze trucker's preferences from history
	routeCount := make(map[string]int)
	totalEarnings := make(map[string]float64)

	for _, booking := range bookings {
		if booking.Status == models.BookingStatusDelivered {
			load, err := r.store.GetLoad(booking.LoadID)
			if err != nil {
				continue
			}

			route := fmt.Sprintf("%s-%s", load.FromCity, load.ToCity)
			routeCount[route]++
			totalEarnings[route] += booking.NetAmount
		}
	}

	// Get all route analytics
	allRoutes, err := r.AnalyzeRoutes()
	if err != nil {
		return nil, err
	}

	// Score routes based on trucker's history and market data
	recommendations := []RouteAnalytics{}

	for _, route := range allRoutes {
		// Calculate personalized score
		personalScore := 0.0

		// Factor 1: Historical preference (40%)
		if count, exists := routeCount[route.Route]; exists {
			personalScore += float64(count) * 0.4
		}

		// Factor 2: Profitability (30%)
		personalScore += route.Profitability * 0.3

		// Factor 3: Load frequency (20%)
		personalScore += float64(route.LoadFrequency) * 0.2

		// Factor 4: Average price (10%)
		personalScore += (route.AveragePrice / 10000) * 0.1

		route.Profitability = personalScore // Override with personalized score
		recommendations = append(recommendations, route)
	}

	// Sort by personalized score
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Profitability > recommendations[j].Profitability
	})

	// Return top 5 recommendations
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	return recommendations, nil
}

// SendRouteSuggestions sends route suggestions to truckers
func (r *RouteSuggestionService) SendRouteSuggestions() error {
	log.Println("Sending route suggestions to truckers...")

	templateService := NewTemplateService(r.twilioService)

	// Get all active truckers
	truckers, err := r.store.GetAllTruckers()
	if err != nil {
		return fmt.Errorf("failed to get truckers: %v", err)
	}

	sentCount := 0

	for _, trucker := range truckers {
		// Skip inactive or suspended truckers
		if !trucker.IsActive || trucker.IsSuspended {
			continue
		}

		// Get recommendations for this trucker
		recommendations, err := r.GetTruckerRecommendations(trucker.TruckerID)
		if err != nil {
			log.Printf("Failed to get recommendations for trucker %s: %v", trucker.TruckerID, err)
			continue
		}

		if len(recommendations) == 0 {
			continue
		}

		// Send top 3 route suggestions
		routes := []string{}
		earnings := []string{}

		for i := 0; i < 3 && i < len(recommendations); i++ {
			routes = append(routes, recommendations[i].Route)
			earnings = append(earnings, fmt.Sprintf("₹%.0f", recommendations[i].AveragePrice))
		}

		params := map[string]string{
			"name":              trucker.Name,
			"route_1":           routes[0],
			"earnings_1":        earnings[0],
			"route_2":           "",
			"earnings_2":        "",
			"route_3":           "",
			"earnings_3":        "",
			"total_suggestions": fmt.Sprintf("%d", len(recommendations)),
		}

		// Add second and third routes if available
		if len(routes) > 1 {
			params["route_2"] = routes[1]
			params["earnings_2"] = earnings[1]
		}
		if len(routes) > 2 {
			params["route_3"] = routes[2]
			params["earnings_3"] = earnings[2]
		}

		err = templateService.SendTemplate(trucker.Phone, "route_suggestion", params)
		if err != nil {
			log.Printf("Failed to send route suggestions to %s: %v", trucker.Phone, err)
			continue
		}

		sentCount++
		log.Printf("Route suggestions sent to %s", trucker.Name)
	}

	log.Printf("Route suggestions sent to %d truckers", sentCount)
	return nil
}

// GetCurrentLoadDemand analyzes current load demand by route
func (r *RouteSuggestionService) GetCurrentLoadDemand() (map[string]int, error) {
	// Get all available loads
	loads, err := r.store.GetAvailableLoads()
	if err != nil {
		return nil, err
	}

	demand := make(map[string]int)

	for _, load := range loads {
		route := fmt.Sprintf("%s-%s", load.FromCity, load.ToCity)
		demand[route]++
	}

	return demand, nil
}

// PredictHighDemandRoutes predicts routes with high demand
func (r *RouteSuggestionService) PredictHighDemandRoutes() ([]string, error) {
	// Get current demand
	currentDemand, err := r.GetCurrentLoadDemand()
	if err != nil {
		return nil, err
	}

	// Get historical analytics
	historicalRoutes, err := r.AnalyzeRoutes()
	if err != nil {
		return nil, err
	}

	// Combine current and historical data
	type routeScore struct {
		route string
		score float64
	}

	scores := []routeScore{}

	for _, hist := range historicalRoutes {
		score := routeScore{
			route: hist.Route,
			score: float64(hist.LoadFrequency) * 0.6, // 60% weight to historical
		}

		// Add current demand weight
		if demand, exists := currentDemand[hist.Route]; exists {
			score.score += float64(demand) * 0.4 // 40% weight to current
		}

		scores = append(scores, score)
	}

	// Sort by score
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Return top routes
	highDemandRoutes := []string{}
	for i := 0; i < 10 && i < len(scores); i++ {
		highDemandRoutes = append(highDemandRoutes, scores[i].route)
	}

	return highDemandRoutes, nil
}

// GetAlternativeRoutes suggests alternative routes when primary route has no loads
func (r *RouteSuggestionService) GetAlternativeRoutes(fromCity, toCity string) ([]RouteAnalytics, error) {
	// Major city connections for route alternatives
	cityConnections := map[string][]string{
		"DELHI":     {"GURGAON", "NOIDA", "FARIDABAD", "GHAZIABAD"},
		"MUMBAI":    {"THANE", "NAVI MUMBAI", "PUNE"},
		"BANGALORE": {"MYSORE", "TUMKUR", "HOSUR"},
		"CHENNAI":   {"KANCHIPURAM", "VELLORE", "TIRUVALLUR"},
		"KOLKATA":   {"HOWRAH", "DURGAPUR", "ASANSOL"},
		"HYDERABAD": {"SECUNDERABAD", "WARANGAL", "NIZAMABAD"},
		"PUNE":      {"MUMBAI", "NASHIK", "SOLAPUR"},
		"AHMEDABAD": {"GANDHINAGAR", "VADODARA", "RAJKOT"},
	}

	alternatives := []RouteAnalytics{}

	// Get nearby cities for origin
	nearbyOrigins := []string{strings.ToUpper(fromCity)}
	if cities, exists := cityConnections[strings.ToUpper(fromCity)]; exists {
		nearbyOrigins = append(nearbyOrigins, cities...)
	}

	// Get nearby cities for destination
	nearbyDestinations := []string{strings.ToUpper(toCity)}
	if cities, exists := cityConnections[strings.ToUpper(toCity)]; exists {
		nearbyDestinations = append(nearbyDestinations, cities...)
	}

	// Get all available loads
	loads, err := r.store.GetAvailableLoads()
	if err != nil {
		return nil, err
	}

	// Find loads on alternative routes
	routeLoads := make(map[string][]*models.Load)

	for _, load := range loads {
		loadFrom := strings.ToUpper(load.FromCity)
		loadTo := strings.ToUpper(load.ToCity)

		// Check if load matches any alternative route
		for _, origin := range nearbyOrigins {
			for _, destination := range nearbyDestinations {
				if loadFrom == origin && loadTo == destination {
					route := fmt.Sprintf("%s-%s", load.FromCity, load.ToCity)
					routeLoads[route] = append(routeLoads[route], load)
				}
			}
		}
	}

	// Create route analytics for alternatives
	for route, loads := range routeLoads {
		if len(loads) > 0 {
			avgPrice := 0.0
			for _, load := range loads {
				avgPrice += load.Price
			}
			avgPrice = avgPrice / float64(len(loads))

			parts := strings.Split(route, "-")
			analytics := RouteAnalytics{
				Route:         route,
				FromCity:      parts[0],
				ToCity:        parts[1],
				AveragePrice:  avgPrice,
				LoadFrequency: len(loads),
				Profitability: avgPrice * float64(len(loads)) / 1000,
			}

			alternatives = append(alternatives, analytics)
		}
	}

	// Sort by number of available loads
	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].LoadFrequency > alternatives[j].LoadFrequency
	})

	return alternatives, nil
}

// CalculateRouteDistance estimates distance between cities (simplified)
func (r *RouteSuggestionService) CalculateRouteDistance(fromCity, toCity string) float64 {
	// In production, use actual distance API or database
	// This is a simplified distance matrix for major cities
	distances := map[string]float64{
		"DELHI-MUMBAI":        1400,
		"DELHI-BANGALORE":     2150,
		"DELHI-KOLKATA":       1500,
		"DELHI-CHENNAI":       2200,
		"MUMBAI-BANGALORE":    980,
		"MUMBAI-CHENNAI":      1330,
		"MUMBAI-KOLKATA":      1970,
		"BANGALORE-CHENNAI":   350,
		"BANGALORE-HYDERABAD": 570,
		"CHENNAI-KOLKATA":     1670,
	}

	route1 := fmt.Sprintf("%s-%s", strings.ToUpper(fromCity), strings.ToUpper(toCity))
	route2 := fmt.Sprintf("%s-%s", strings.ToUpper(toCity), strings.ToUpper(fromCity))

	if distance, exists := distances[route1]; exists {
		return distance
	}
	if distance, exists := distances[route2]; exists {
		return distance
	}

	// Default estimate based on different cities
	return 500.0
}

// GetOptimalLoadCombinations finds optimal multi-load combinations
func (r *RouteSuggestionService) GetOptimalLoadCombinations(truckerID string) ([][]string, error) {
	// Get trucker details
	trucker, err := r.store.GetTruckerByID(truckerID)
	if err != nil {
		return nil, err
	}

	// Get available loads
	loads, err := r.store.GetAvailableLoads()
	if err != nil {
		return nil, err
	}

	// Group loads by compatible routes
	combinations := [][]string{}

	// Simple algorithm: Find loads that can be picked up and delivered in sequence
	for i, load1 := range loads {
		for j := i + 1; j < len(loads); j++ {
			load2 := loads[j]

			// Check if load2 pickup is near load1 delivery
			if strings.EqualFold(load1.ToCity, load2.FromCity) {
				// Check if combined weight is within truck capacity
				if load1.Weight+load2.Weight <= trucker.Capacity {
					combinations = append(combinations, []string{load1.LoadID, load2.LoadID})
				}
			}
		}
	}

	// Limit to top 5 combinations
	if len(combinations) > 5 {
		combinations = combinations[:5]
	}

	return combinations, nil
}

// ScheduleRouteSuggestions sets up scheduled route suggestions
func (r *RouteSuggestionService) ScheduleRouteSuggestions() {
	// Send route suggestions every Monday and Thursday at 9 AM
	go func() {
		for {
			now := time.Now()

			// Calculate next run time
			var nextRun time.Time
			weekday := now.Weekday()

			// Find next Monday or Thursday
			daysUntilNext := 0
			if weekday <= time.Monday {
				daysUntilNext = int(time.Monday - weekday)
			} else if weekday <= time.Thursday {
				daysUntilNext = int(time.Thursday - weekday)
			} else {
				daysUntilNext = int(time.Monday) + 7 - int(weekday)
			}

			if daysUntilNext == 0 && now.Hour() >= 9 {
				// If it's Monday/Thursday after 9 AM, schedule for next occurrence
				if weekday == time.Monday {
					daysUntilNext = 3 // Thursday
				} else {
					daysUntilNext = 4 // Next Monday
				}
			}

			nextRun = time.Date(now.Year(), now.Month(), now.Day()+daysUntilNext, 9, 0, 0, 0, now.Location())
			duration := nextRun.Sub(now)

			log.Printf("Next route suggestions scheduled in %v", duration)
			time.Sleep(duration)

			// Send suggestions
			if err := r.SendRouteSuggestions(); err != nil {
				log.Printf("Error sending route suggestions: %v", err)
			}
		}
	}()
}

// GetRouteInsights provides detailed insights for a specific route
func (r *RouteSuggestionService) GetRouteInsights(fromCity, toCity string) (*RouteInsights, error) {
	route := fmt.Sprintf("%s-%s", fromCity, toCity)

	// Get historical data
	analytics, err := r.AnalyzeRoutes()
	if err != nil {
		return nil, err
	}

	var routeData *RouteAnalytics
	for _, data := range analytics {
		if data.Route == route {
			routeData = &data
			break
		}
	}

	if routeData == nil {
		return &RouteInsights{
			Route:          route,
			DataAvailable:  false,
			Recommendation: "No historical data available for this route",
		}, nil
	}

	// Calculate insights
	distance := r.CalculateRouteDistance(fromCity, toCity)
	pricePerKm := routeData.AveragePrice / distance

	insights := &RouteInsights{
		Route:           route,
		DataAvailable:   true,
		Distance:        distance,
		AveragePrice:    routeData.AveragePrice,
		PricePerKm:      pricePerKm,
		LoadFrequency:   routeData.LoadFrequency,
		AverageDuration: routeData.AverageDuration,
		BestDays:        r.analyzeBestDays(route),
		PeakSeasons:     r.analyzePeakSeasons(route),
		Recommendation:  r.generateRecommendation(routeData, pricePerKm),
	}

	return insights, nil
}

// RouteInsights contains detailed route analysis
type RouteInsights struct {
	Route           string   `json:"route"`
	DataAvailable   bool     `json:"data_available"`
	Distance        float64  `json:"distance_km"`
	AveragePrice    float64  `json:"average_price"`
	PricePerKm      float64  `json:"price_per_km"`
	LoadFrequency   int      `json:"load_frequency"`
	AverageDuration float64  `json:"average_duration_hours"`
	BestDays        []string `json:"best_days"`
	PeakSeasons     []string `json:"peak_seasons"`
	Recommendation  string   `json:"recommendation"`
}

// Helper methods for route insights
func (r *RouteSuggestionService) analyzeBestDays(route string) []string {
	// In production, analyze actual booking data by day
	// For now, return common high-demand days
	return []string{"Monday", "Thursday", "Friday"}
}

func (r *RouteSuggestionService) analyzePeakSeasons(route string) []string {
	// In production, analyze seasonal patterns
	// For now, return common peak seasons
	return []string{"Oct-Dec", "Feb-Apr"}
}

func (r *RouteSuggestionService) generateRecommendation(routeData *RouteAnalytics, pricePerKm float64) string {
	if routeData.LoadFrequency > 20 && pricePerKm > 25 {
		return "High-demand profitable route. Prioritize this route for maximum earnings."
	} else if routeData.LoadFrequency > 10 {
		return "Regular route with steady demand. Good for consistent earnings."
	} else if pricePerKm > 30 {
		return "Premium pricing route. Less frequent but high-value loads."
	} else {
		return "Emerging route. Monitor for increasing demand."
	}
}
