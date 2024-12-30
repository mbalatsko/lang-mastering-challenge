package users

import (
	"api-server/pkg/utils"
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const TokenExpiration = time.Hour * 72

var ErrEmailAlreadyExists = errors.New("user with such email already exists")

type UserService struct {
	Repo      *UserRepo
	JwtSecret string
}

func NewUserService(repo *UserRepo) *UserService {
	jwtSecret := utils.MustGetenv("JWT_SECRET")
	return &UserService{Repo: repo, JwtSecret: jwtSecret}
}

func (s *UserService) Register(ctx context.Context, cred *UserCredentials) error {
	emailExists, err := s.Repo.EmailExists(ctx, cred.Email)
	if emailExists {
		return ErrEmailAlreadyExists
	}
	if err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil
	}
	return s.Repo.Create(ctx, cred.Email, string(passwordHash))
}

func (s *UserService) compareHashAndPassword(hashedPassword string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func (s *UserService) Login(ctx context.Context, cred *UserCredentials) (string, bool, error) {
	user, found, err := s.Repo.GetByEmail(ctx, cred.Email)
	if err != nil {
		return "", false, err
	}
	if !found {
		return "", false, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"exp":   time.Now().Add(TokenExpiration).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.JwtSecret))
	if err != nil {
		return "", false, err
	}
	return tokenString, s.compareHashAndPassword(user.PasswordHash, cred.Password), nil
}
