package handlers

import (
	"api-server/internal/app/middlewares"
	"api-server/internal/domain/models"
	"api-server/internal/domain/services"
	"api-server/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleRegistration(userService *services.UsersService) func(*gin.Context) {
	return func(c *gin.Context) {
		var cred models.UserCredentials
		if err := c.ShouldBindBodyWithJSON(&cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userService.Register(c.Request.Context(), cred.Email, cred.Password)
		if err == services.ErrEmailAlreadyExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusCreated)
	}
}

func HandleLogin(userService *services.UsersService) func(*gin.Context) {
	return func(c *gin.Context) {
		var cred models.UserCredentials
		if err := c.ShouldBindBodyWithJSON(&cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token, success, err := userService.Login(c.Request.Context(), cred.Email, cred.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !success {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}

func HandleWhoAmI(userService *services.UsersService, jwtAuth *middlewares.JwtAuthenticator) func(*gin.Context) {
	return func(c *gin.Context) {
		userData, err := utils.GetUserFromCtx(c, jwtAuth.AuthCtxKey)
		if err != nil {
			return
		}

		c.JSON(http.StatusOK, userData)
	}
}
