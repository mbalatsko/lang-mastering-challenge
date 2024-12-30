package routes

import (
	"api-server/internal/app/handlers"
	"api-server/internal/app/middlewares"
	"api-server/internal/domain/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupDefaultRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	return r
}

func RegisterAuthRoutes(r *gin.Engine, jwtAuth *middlewares.JwtAuthenticator, usersService *services.UsersService) {
	g := r.Group("/auth")
	g.POST("/register", handlers.HandleRegistration(usersService))
	g.POST("/login", handlers.HandleLogin(usersService))
	g.GET("/whoami", jwtAuth.Handler, handlers.HandleWhoAmI(usersService, jwtAuth))
}

func RegisterTasksRoutes(r *gin.Engine, jwtAuth *middlewares.JwtAuthenticator, tasksService *services.TasksService) {
	g := r.Group("/tasks")
	g.GET("/", jwtAuth.Handler, handlers.HandleListTasks(tasksService, jwtAuth))
	g.POST("/", jwtAuth.Handler, handlers.HandleCreateTask(tasksService, jwtAuth))
}
