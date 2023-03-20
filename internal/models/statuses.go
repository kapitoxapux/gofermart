package models

import "time"

type Status struct {
	ID        uint64    `gorm:"primary_key" json:"id"`
	Order     int       `gorm:"index:order;unique" json:"order"`
	Status    string    `gorm:"not null;default:REGISTERED" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
