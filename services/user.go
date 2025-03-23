package services

import (
	"context"
	"time"

	db "idiomatic-go/database"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserService struct {
	queries *db.Queries
	logger  *logrus.Logger
}

func NewUserService(queries *db.Queries, logger *logrus.Logger) *UserService {
	return &UserService{
		queries: queries,
		logger:  logger,
	}
}

func (s *UserService) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	// Hash password before storing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithError(err).Error("failed to hash password")
		return db.User{}, err
	}
	params.PasswordHash = string(hashedPassword)

	user, err := s.queries.CreateUser(ctx, params)
	if err != nil {
		s.logger.WithError(err).Error("failed to create user")
		return db.User{}, err
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (db.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.WithError(err).Warn("user not found")
		return db.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.WithError(err).Warn("invalid password")
		return db.User{}, err
	}

	return user, nil
}
