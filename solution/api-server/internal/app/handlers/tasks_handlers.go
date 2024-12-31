package handlers

import (
	"api-server/internal/app/middlewares"
	"api-server/internal/domain/models"
	"api-server/internal/domain/services"
	"api-server/internal/utils"
	"net/http"
	"strconv"

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

func HandleDeleteTask(tasksService *services.TasksService, jwtAuth *middlewares.JwtAuthenticator) func(*gin.Context) {
	return func(c *gin.Context) {
		userData, err := utils.GetUserFromCtx(c, jwtAuth.AuthCtxKey)
		if err != nil {
			return
		}
		taskIdParam := c.Param("id")
		taskId, err := strconv.Atoi(taskIdParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID in URL path"})
			return
		}

		err = tasksService.DeleteById(c, taskId, userData.Id)
		if err == services.ErrTaskDoesNotExist {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err == services.ErrNotOwner {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func HandleUpdateTask(tasksService *services.TasksService, jwtAuth *middlewares.JwtAuthenticator) func(*gin.Context) {
	return func(c *gin.Context) {
		userData, err := utils.GetUserFromCtx(c, jwtAuth.AuthCtxKey)
		if err != nil {
			return
		}

		taskIdParam := c.Param("id")
		taskId, err := strconv.Atoi(taskIdParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID in URL path"})
			return
		}

		var taskStatus models.TaskStatus
		if err := c.ShouldBindBodyWithJSON(&taskStatus); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updatedTask, err := tasksService.UpdateStatus(c, taskId, taskStatus.Status, userData.Id)
		if err == services.ErrTaskDoesNotExist {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err == services.ErrNotOwner {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, updatedTask)
	}
}
