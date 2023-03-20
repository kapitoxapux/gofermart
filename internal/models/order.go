package models

import "time"

type Order struct {
	ID          uint64    `gorm:"primary_key" json:"id"`
	UserId      uint64    `gorm:"index:user_id;" json:"user_id"`
	OrderNumber int       `gorm:"index:order_number;unique" json:"order_number"`
	Status      string    `gorm:"not null" json:"status"`
	Accrual     float64   `gorm:"type:float;default:0;not null" json:"accrual"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
