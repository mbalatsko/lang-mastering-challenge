package routes

import (
	"api-server/app/handlers"
	"api-server/app/middlewares"
	"api-server/domain/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func SetupDefaultRouter() *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		startTime := time.Now()

		c.Next()

		latency := time.Since(startTime)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		// Log using log.WithFields
		log.WithFields(log.Fields{
			"status_code":     statusCode,
			"latency_seconds": latency.Seconds(),
			"client_ip":       clientIP,
			"method":          method,
			"path":            path,
		}).Info("Request completed")
	})
	r.Use(gin.Recovery())

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
