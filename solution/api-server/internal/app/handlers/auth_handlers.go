package handlers

import (
	"api-server/internal/app/middlewares"
	"api-server/internal/domain/models"
	"api-server/internal/domain/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleRegistration(userService *services.UserService) func(*gin.Context) {
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

func HandleLogin(userService *services.UserService) func(*gin.Context) {
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

func HandleWhoAmI(userService *services.UserService, jwtAuth *middlewares.JwtAuthenticator) func(*gin.Context) {
	return func(c *gin.Context) {
		userDataI, ok := c.Get(jwtAuth.AuthCtxKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "user is not provided by middleware"})
			return
		}

		userData, ok := userDataI.(models.UserData)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "wrong user type provided by middleware"})
			return
		}

		c.JSON(http.StatusOK, userData)
	}
}
