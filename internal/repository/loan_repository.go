package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"libratrack/internal/model"
)

var ErrCopyUnavailable = errors.New("copy is not available")

type LoanRepository interface {
	Create(context.Context, uint, uint) (model.Loan, error)
	ListByUser(context.Context, uint) ([]model.Loan, error)
	Return(context.Context, uint) (model.Loan, error)
}

type gormLoanRepository struct {
	db *gorm.DB
}

func NewGormLoanRepository(db *gorm.DB) LoanRepository {
	return &gormLoanRepository{db: db}
}

func (r *gormLoanRepository) Create(ctx context.Context, userID, copyID uint) (model.Loan, error) {
	loan := model.Loan{UserID: userID, CopyID: copyID, LoanedAt: time.Now().UTC(), DueAt: time.Now().UTC().AddDate(0, 0, 14)}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&model.Copy{}).Where("id = ? AND status = ?", copyID, "available").Update("status", "borrowed")
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return ErrCopyUnavailable
		}
		return tx.Create(&loan).Error
	})
	return loan, err
}

func (r *gormLoanRepository) ListByUser(ctx context.Context, userID uint) ([]model.Loan, error) {
	loans := make([]model.Loan, 0)
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&loans).Error
	return loans, err
}

func (r *gormLoanRepository) Return(ctx context.Context, id uint) (model.Loan, error) {
	var loan model.Loan
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&loan, id).Error; err != nil {
			return err
		}
		if loan.ReturnedAt != nil {
			return gorm.ErrRecordNotFound
		}
		now := time.Now().UTC()
		result := tx.Model(&model.Loan{}).Where("id = ? AND returned_at IS NULL", id).Update("returned_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&model.Copy{}).Where("id = ?", loan.CopyID).Update("status", "available").Error; err != nil {
			return err
		}
		loan.ReturnedAt = &now
		return nil
	})
	return loan, err
}
