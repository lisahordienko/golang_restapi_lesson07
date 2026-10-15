package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"libratrack/internal/model"
	"libratrack/internal/repository"
)

type LoanService struct {
	loans repository.LoanRepository
}

func NewLoanService(loans repository.LoanRepository) *LoanService {
	return &LoanService{loans: loans}
}

func (s *LoanService) Create(ctx context.Context, userID, copyID uint) (model.Loan, error) {
	loan, err := s.loans.Create(ctx, userID, copyID)
	if errors.Is(err, repository.ErrCopyUnavailable) {
		return model.Loan{}, ErrConflict
	}
	return loan, err
}

func (s *LoanService) Mine(ctx context.Context, userID uint) ([]model.Loan, error) {
	return s.loans.ListByUser(ctx, userID)
}

func (s *LoanService) Return(ctx context.Context, loanID uint) (model.Loan, error) {
	loan, err := s.loans.Return(ctx, loanID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Loan{}, ErrNotFound
	}
	return loan, err
}
