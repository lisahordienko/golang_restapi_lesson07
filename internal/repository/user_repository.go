package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"libratrack/internal/auth"
	"libratrack/internal/model"
)

type gormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) auth.UserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *gormUserRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, auth.ErrUserNotFound
	}
	return user, err
}

func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	return user, err
}

func (r *gormUserRepository) List(ctx context.Context) ([]model.User, error) {
	users := make([]model.User, 0)
	err := r.db.WithContext(ctx).Select("id", "email", "role", "created_at", "updated_at").Order("id ASC").Find(&users).Error
	return users, err
}

func (r *gormUserRepository) UpdateEmail(ctx context.Context, id uint, email string) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("email", email)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *gormUserRepository) UpdateRole(ctx context.Context, id uint, role string) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("role", role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var _ auth.UserRepository = (*gormUserRepository)(nil)
