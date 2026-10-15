package model

import "time"

type Loan struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"not null;index"`
	CopyID     uint       `json:"copy_id" gorm:"not null;index"`
	LoanedAt   time.Time  `json:"loaned_at" gorm:"not null"`
	DueAt      time.Time  `json:"due_at" gorm:"not null"`
	ReturnedAt *time.Time `json:"returned_at"`
}
