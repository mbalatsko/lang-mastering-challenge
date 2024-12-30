package main

import (
	"api-server/internal/app/middlewares"
	"api-server/internal/app/routes"
	"api-server/internal/db"
	"api-server/internal/domain/repos"
	"api-server/internal/domain/services"
	"api-server/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Services struct {
	TokenProvider *services.JwtTokenProvider
	UserService   *services.UserService
	UserRepo      *repos.UserRepo
}

func SetupDependencies(conn *pgxpool.Pool) *Services {
	tp := services.NewJwtTokenProvider()

	userRepo := repos.NewUserRepo(conn)
	userService := services.NewUserService(userRepo, tp)

	return &Services{
		TokenProvider: tp,
		UserService:   userService,
		UserRepo:      userRepo,
	}
}

func main() {
	// Setup DB connection
	conn := db.ConnectDB()

	// Setup services
	services := SetupDependencies(conn)

	// Register validators
	utils.RegisterValidators()

	// Setup Auth middleware
	jwtAuth := middlewares.NewJwtAuthenticator(services.TokenProvider, services.UserRepo)

	// Register all app routes
	r := routes.SetupDefaultRouter()
	routes.RegisterAuthRoutes(r, jwtAuth, services.UserService)

	r.Run(":9090")
}
