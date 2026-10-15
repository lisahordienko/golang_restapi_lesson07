package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"libratrack/internal/model"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("email already registered")
	ErrJWTSecretMissing   = errors.New("JWT_SECRET is not configured")
	ErrInvalidRole        = errors.New("invalid role")
)

var dummyPasswordHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

type UserRepository interface {
	Create(context.Context, *model.User) error
	FindByEmail(context.Context, string) (model.User, error)
	FindByID(context.Context, uint) (model.User, error)
	List(context.Context) ([]model.User, error)
	UpdateEmail(context.Context, uint, string) error
	UpdateRole(context.Context, uint, string) error
}

type RegisterInput struct {
	Email    string
	Password string
}

type Service struct {
	users UserRepository
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func NewService(users UserRepository) *Service {
	return &Service{users: users}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (model.User, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Email == "" || !strings.Contains(input.Email, "@") || len(input.Password) < 8 || len(input.Password) > 72 {
		return model.User{}, fmt.Errorf("invalid registration fields")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	user := model.User{Email: input.Email, PasswordHash: string(passwordHash), Role: "reader"}
	if err := s.users.Create(ctx, &user); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return model.User{}, ErrDuplicateEmail
		}
		return model.User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (model.User, string, error) {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	passwordHash := []byte(user.PasswordHash)
	if errors.Is(err, ErrUserNotFound) || (err == nil && user.PasswordHash == "") {
		passwordHash = dummyPasswordHash
	}
	passwordErr := bcrypt.CompareHashAndPassword(passwordHash, []byte(password))
	if errors.Is(err, ErrUserNotFound) || (err == nil && user.PasswordHash == "") || (err == nil && passwordErr != nil) {
		return model.User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return model.User{}, "", err
	}
	token, err := IssueToken(user)
	if err != nil {
		return model.User{}, "", err
	}
	return user, token, nil
}

func IssueToken(user model.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", ErrJWTSecretMissing
	}
	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseToken(rawToken string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, ErrJWTSecretMissing
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || token == nil || !token.Valid || claims.UserID == 0 || claims.Role == "" || claims.ExpiresAt == nil {
		return nil, ErrInvalidCredentials
	}
	return claims, nil
}

func (s *Service) User(ctx context.Context, id uint) (model.User, error) {
	return s.users.FindByID(ctx, id)
}

func (s *Service) Users(ctx context.Context) ([]model.User, error) {
	return s.users.List(ctx)
}

func (s *Service) UpdateProfile(ctx context.Context, id uint, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email")
	}
	return s.users.UpdateEmail(ctx, id, email)
}

func (s *Service) UpdateRole(ctx context.Context, id uint, role string) error {
	if role != "reader" && role != "librarian" && role != "admin" {
		return ErrInvalidRole
	}
	return s.users.UpdateRole(ctx, id, role)
}
