package models

import (
	"time"

	"gorm.io/gorm"
)

type TruckerStats struct {
	gorm.Model
	TruckerID      string     `json:"trucker_id" gorm:"uniqueIndex"`
	CompletedTrips int        `json:"completed_trips"`
	TotalEarnings  float64    `json:"total_earnings"`
	AverageRating  float64    `json:"average_rating"`
	OnTimeDelivery float64    `json:"on_time_delivery_rate"`
	LastActiveAt   *time.Time `json:"last_active_at"`
}

type ShipperStats struct {
	gorm.Model
	ShipperID      string  `json:"shipper_id" gorm:"uniqueIndex"`
	TotalLoads     int     `json:"total_loads"`
	ActiveLoads    int     `json:"active_loads"`
	CompletedLoads int     `json:"completed_loads"`
	TotalSpent     float64 `json:"total_spent"`
}
