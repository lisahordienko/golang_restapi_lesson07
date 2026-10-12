package model

import "time"

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"not null;uniqueIndex"`
	Role      string    `json:"role" gorm:"not null;default:member"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
