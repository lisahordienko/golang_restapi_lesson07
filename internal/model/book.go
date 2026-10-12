package model

import "time"

type Book struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Title      string    `json:"title" gorm:"not null"`
	ISBN       string    `json:"isbn" gorm:"not null;uniqueIndex"`
	CategoryID uint      `json:"category_id" gorm:"not null;index"`
	Category   Category  `json:"category" gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Authors    []Author  `json:"authors" gorm:"many2many:book_authors;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Copies     []Copy    `json:"copies,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
