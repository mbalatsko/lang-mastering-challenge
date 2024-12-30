package services

import (
	"api-server/internal/domain/models"
	"api-server/internal/domain/repos"
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("user with such email already exists")
	ErrUserNotFound       = errors.New("user with given email is not found")
)

type UserService struct {
	Repo          *repos.UserRepo
	TokenProvider *JwtTokenProvider
}

func NewUserService(repo *repos.UserRepo, tp *JwtTokenProvider) *UserService {
	return &UserService{Repo: repo, TokenProvider: tp}
}

func (s *UserService) Register(ctx context.Context, email string, password string) error {
	emailExists, err := s.Repo.EmailExists(ctx, email)
	if emailExists {
		return ErrEmailAlreadyExists
	}
	if err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil
	}

	_, err = s.Repo.Create(ctx, email, string(passwordHash))
	return err
}

func (s *UserService) compareHashAndPassword(hashedPassword string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func (s *UserService) EmailExists(ctx context.Context, email string) bool {
	emailExists, _ := s.Repo.EmailExists(ctx, email)
	return emailExists
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (models.UserData, error) {
	user, found, err := s.Repo.GetByEmail(ctx, email)
	if err != nil {
		return user, err
	}
	if !found {
		return user, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, email string, password string) (string, bool, error) {
	user, found, err := s.Repo.GetByEmail(ctx, email)
	if err != nil {
		return "", false, err
	}

	if !found {
		return "", false, nil
	}

	tokenString, err := s.TokenProvider.Provide(email)
	if err != nil {
		return "", false, err
	}
	return tokenString, s.compareHashAndPassword(user.PasswordHash, password), nil
}
