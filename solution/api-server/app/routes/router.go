package routes

import (
	"api-server/app/handlers"
	"api-server/app/middlewares"
	"api-server/domain/services"
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

func RegisterAuthRoutes(r *gin.Engine, jwtHeaderAuth *middlewares.JwtHeaderAuthenticator, usersService *services.UsersService) {
	g := r.Group("/auth")
	g.POST("/register", handlers.HandleRegistration(usersService))
	g.POST("/login", handlers.HandleLogin(usersService))
	g.GET("/whoami", jwtHeaderAuth.Handler, handlers.HandleWhoAmI(usersService, jwtHeaderAuth))
}

func RegisterTasksRoutes(r *gin.Engine, jwtHeaderAuth *middlewares.JwtHeaderAuthenticator, tasksService *services.TasksService) {
	g := r.Group("/tasks")
	g.GET("/", jwtHeaderAuth.Handler, handlers.HandleListTasks(tasksService, jwtHeaderAuth))
	g.POST("/", jwtHeaderAuth.Handler, handlers.HandleCreateTask(tasksService, jwtHeaderAuth))

	g.DELETE("/:id", jwtHeaderAuth.Handler, handlers.HandleDeleteTask(tasksService, jwtHeaderAuth))
	g.PATCH("/:id", jwtHeaderAuth.Handler, handlers.HandleUpdateTask(tasksService, jwtHeaderAuth))
}

func RegisterDashboardRoute(r *gin.Engine, jwtCookieAuth *middlewares.JwtCookieAuthenticator, tasksService *services.TasksService) {
	r.GET("/dashboard/", jwtCookieAuth.Handler, handlers.HandleDashboard(tasksService, jwtCookieAuth))
}
