package handlers

import (
	"api-server/internal/app/middlewares"
	"api-server/internal/domain/models"
	"api-server/internal/domain/services"
	"api-server/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleListTasks(tasksService *services.TasksService, jwtAuth *middlewares.JwtAuthenticator) func(*gin.Context) {
	return func(c *gin.Context) {
		userData, err := utils.GetUserFromCtx(c, jwtAuth.AuthCtxKey)
		if err != nil {
			return
		}

		tasks, err := tasksService.ListByUserId(c, userData.Id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tasks)
	}
}

func HandleCreateTask(tasksService *services.TasksService, jwtAuth *middlewares.JwtAuthenticator) func(*gin.Context) {
	return func(c *gin.Context) {
		userData, err := utils.GetUserFromCtx(c, jwtAuth.AuthCtxKey)
		if err != nil {
			return
		}

		var taskCreate models.TaskCreate
		if err := c.ShouldBindBodyWithJSON(&taskCreate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		task, err := tasksService.Create(c, taskCreate, userData.Id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, task)
	}
}
