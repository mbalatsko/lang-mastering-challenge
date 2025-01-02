package handlers

import (
	"api-server/app/middlewares"
	"api-server/domain/models"
	"api-server/domain/services"
	"api-server/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleRegistration(userService *services.UsersService) func(*gin.Context) {
	return func(c *gin.Context) {
		var userRegister models.UserRegister
		if err := c.ShouldBindBodyWithJSON(&userRegister); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userService.Register(c.Request.Context(), userRegister.Email, userRegister.Password)
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
		var cred models.UserLogin
		if err := c.ShouldBindBodyWithJSON(&cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		token, err := userService.Login(c.Request.Context(), cred.Email, cred.Password)
		if err == services.ErrUserNotFound || err == services.ErrIncorrectPassword {
			c.Status(http.StatusUnauthorized)
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
