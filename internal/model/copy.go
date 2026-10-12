package model

import "time"

type Copy struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BookID      uint      `json:"book_id" gorm:"not null;index"`
	InventoryNo string    `json:"inventory_no" gorm:"not null;uniqueIndex"`
	Status      string    `json:"status" gorm:"not null;default:available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
