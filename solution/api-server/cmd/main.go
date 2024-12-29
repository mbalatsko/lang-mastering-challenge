package main

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"api-server/pkg/db"
)

type UserCredentials struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,strongpass"`
}

var strongPasswordValidator validator.Func = func(fl validator.FieldLevel) bool {
	password, ok := fl.Field().Interface().(string)
	if ok {
		var (
			lengthValid      = len(password) >= 8 && len(password) <= 20 // 8-20 characters
			lowercaseRegex   = regexp.MustCompile(`[a-z]`)               // At least one lowercase
			uppercaseRegex   = regexp.MustCompile(`[A-Z]`)               // At least one uppercase
			digitRegex       = regexp.MustCompile(`\d`)                  // At least one digit
			specialCharRegex = regexp.MustCompile(`[!@#$%^&*()-_+=<>?]`) // At least one special character
		)

		return lengthValid &&
			lowercaseRegex.MatchString(password) &&
			uppercaseRegex.MatchString(password) &&
			digitRegex.MatchString(password) &&
			specialCharRegex.MatchString(password)
	}
	return true
}

type UserData struct {
	Id           int
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type UserRepository struct {
	Db *pgxpool.Pool
}

func (repo *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var emailExists bool

	err := repo.Db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&emailExists)
	if err != nil {
		return false, err
	}
	return emailExists, nil
}

func (repo *UserRepository) Create(ctx context.Context, email string, passwordHash string) error {
	_, err := repo.Db.Exec(ctx, "INSERT INTO users (email, password_hash) VALUES ($1, $2)", email, passwordHash)
	return err
}

func (repo *UserRepository) GetByEmail(ctx context.Context, email string) (user UserData, found bool, err error) {
	rows, err := repo.Db.Query(ctx, "SELECT id, email, password_hash, created_at FROM users WHERE email = $1", email)
	if err != nil {
		return UserData{}, false, err
	}

	user, err = pgx.CollectOneRow(rows, pgx.RowToStructByPos[UserData])
	if errors.Is(err, pgx.ErrNoRows) {
		return UserData{}, false, nil
	}
	if err != nil {
		return UserData{}, false, err
	}
	return user, true, nil
}

var ErrEmailAlreadyExists = errors.New("user with such email already exists")

type UserService struct {
	Repo *UserRepository
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

func (s *UserService) Login(ctx context.Context, cred *UserCredentials) (bool, error) {
	user, found, err := s.Repo.GetByEmail(ctx, cred.Email)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	return s.compareHashAndPassword(user.PasswordHash, cred.Password), nil
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	dbConnPool := db.ConnectDB()

	userRepo := &UserRepository{Db: dbConnPool}
	userService := &UserService{Repo: userRepo}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("strongpass", strongPasswordValidator)
	}

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.POST("/auth/register", func(c *gin.Context) {
		var cred UserCredentials
		if err := c.ShouldBindBodyWithJSON(&cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userService.Register(c.Request.Context(), &cred)
		if err == ErrEmailAlreadyExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusCreated)
	})

	r.POST("/auth/login", func(c *gin.Context) {
		var cred UserCredentials
		if err := c.ShouldBindBodyWithJSON(&cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		success, err := userService.Login(c.Request.Context(), &cred)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !success {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.Status(http.StatusOK)
	})

	return r
}

func main() {
	r := setupRouter()
	r.Run(":9090")
}
