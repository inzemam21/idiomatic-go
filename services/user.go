package services

import (
	"context"
	"database/sql"
	"time"

	"idiomatic-go/database"
	custom_errors "idiomatic-go/errors"

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
	db     *database.DB // Change to full DB to access transactions
	logger *logrus.Logger
}

func NewUserService(db *database.DB, logger *logrus.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

func (s *UserService) CreateUser(ctx context.Context, params database.CreateUserParams) (database.User, error) {
	var user database.User
	err := s.db.WithTx(ctx, func(queries *database.Queries) error {
		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.PasswordHash), bcrypt.DefaultCost)
		if err != nil {
			s.logger.WithError(err).Error("failed to hash password")
			return custom_errors.ErrInternalServerError
		}
		params.PasswordHash = string(hashedPassword)

		// Create user
		user, err = queries.CreateUser(ctx, params)
		if err != nil {
			s.logger.WithError(err).Error("failed to create user")
			return custom_errors.ErrInternalServerError
		}

		// Create audit log
		auditParams := database.CreateAuditLogParams{
			UserID: user.ID,
			Action: "user_created",
		}
		_, err = queries.CreateAuditLog(ctx, auditParams)
		if err != nil {
			s.logger.WithError(err).Error("failed to create audit log")
			return custom_errors.ErrInternalServerError
		}

		return nil
	})
	if err != nil {
		return database.User{}, err
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (database.User, error) {
	user, err := s.db.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.WithField("email", email).Warn("user not found")
			return database.User{}, custom_errors.ErrUnauthorized
		}
		s.logger.WithError(err).Error("failed to get user")
		return database.User{}, custom_errors.ErrInternalServerError
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.WithField("email", email).Warn("invalid password")
		return database.User{}, custom_errors.ErrUnauthorized
	}

	return user, nil
}
