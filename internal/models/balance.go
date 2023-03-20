package models

import "time"

type Balance struct {
	ID        uint64    `gorm:"primary_key" json:"id"`
	UserId    uint64    `gorm:"index:user_id;" json:"user_id"`
	OrderId   int       `gorm:"index:order_id;unique" json:"order_id"`
	Withdraw  float64   `gorm:"type:float;default:0;not null" json:"withdraw"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
