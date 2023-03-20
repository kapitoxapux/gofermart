package models

import "time"

type User struct {
	ID        uint64    `gorm:"primary_key" json:"id"`
	Login     string    `gorm:"index:login;unique" json:"login"`
	Password  string    `gorm:"not null" json:"password"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
