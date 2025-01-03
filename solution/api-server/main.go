package main

import (
	"api-server/app/logger"
	"api-server/app/middlewares"
	"api-server/app/routes"
	"api-server/db"
	"api-server/domain/repos"
	"api-server/domain/services"
	"api-server/utils"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

type Services struct {
	TokenProvider *services.JwtTokenProvider
	UsersService  *services.UsersService
	UsersRepo     *repos.UsersRepo
	TasksService  *services.TasksService
	TasksRepo     *repos.TasksRepo
}

func SetupDependencies(conn *pgxpool.Pool) *Services {
	tp := services.NewJwtTokenProvider()

	userRepo := repos.NewUsersRepo(conn)
	userService := services.NewUsersService(userRepo, tp)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	return &Services{
		TokenProvider: tp,
		UsersService:  userService,
		UsersRepo:     userRepo,
		TasksService:  tasksService,
		TasksRepo:     tasksRepo,
	}
}

func main() {
	addr := "localhost:9090"
	// Init logging
	logger.InitLogging()

	// Setup DB connection
	conn := db.ConnectDB()

	// Setup deps
	deps := SetupDependencies(conn)

	// Register validators
	utils.RegisterValidators()

	// Setup Auth middleware
	jwtHeaderAuth := middlewares.NewJwtHeaderAuthenticator(deps.TokenProvider, deps.UsersRepo)
	jwtCookieAuth := middlewares.NewJwtCookieAuthenticator(deps.TokenProvider, deps.UsersRepo)

	// Register all app routes
	r := routes.SetupDefaultRouter()
	routes.RegisterAuthRoutes(r, jwtHeaderAuth, deps.UsersService)
	routes.RegisterTasksRoutes(r, jwtHeaderAuth, deps.TasksService)
	routes.RegisterDashboardRoute(r, jwtCookieAuth, deps.TasksService)

	log.WithFields(log.Fields{"host": addr}).Info("Starting server")
	r.Run(addr)
}
