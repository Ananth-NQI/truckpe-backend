package models

import (
	"fmt"

	"gorm.io/gorm"
)

type Shipper struct {
	gorm.Model
	ShipperID   string `gorm:"unique;not null"`
	CompanyName string `gorm:"not null"`
	GSTNumber   string `gorm:"unique;not null"`
	Phone       string `gorm:"unique;not null"`
	Email       string
	Address     string
	City        string
	State       string
	Verified    bool    `gorm:"default:false"`
	Active      bool    `gorm:"default:true"`
	TotalLoads  int     `gorm:"default:0"`
	Rating      float64 `gorm:"default:5.0"`
}

// BeforeCreate generates ShipperID
func (s *Shipper) BeforeCreate(tx *gorm.DB) error {
	if s.ShipperID == "" {
		var count int64
		tx.Model(&Shipper{}).Count(&count)
		s.ShipperID = fmt.Sprintf("SH%05d", count+1)
	}
	return nil
}
