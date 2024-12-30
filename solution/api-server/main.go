package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-server/pkg/db"
	"api-server/pkg/features/users"
)

type Services struct {
	UserService *users.UserService
}

func setupServices(conn *pgxpool.Pool) *Services {
	userRepository := users.NewUserRepo(conn)

	userService := users.NewUserService(userRepository)

	return &Services{
		UserService: userService,
	}
}

func setupDefaultRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	return r
}

func main() {
	// Setup DB connection
	conn := db.ConnectDB()

	// Setup services
	services := setupServices(conn)

	// Register validators
	users.RegisterUsersValidators()

	// Register all app routes
	r := setupDefaultRouter()
	users.RegisterUsersRoutes(r, services.UserService)

	r.Run(":9090")
}
