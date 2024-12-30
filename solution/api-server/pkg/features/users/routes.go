package users

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterUsersRoutes(r *gin.Engine, userService *UserService) {
	r.POST("/auth/register", func(c *gin.Context) {
		var cred UserCredentials
		if err := c.ShouldBindBodyWithJSON(&cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userService.Register(c.Request.Context(), &cred)
		if err == ErrEmailAlreadyExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusCreated)
	})

	r.POST("/auth/login", func(c *gin.Context) {
		var cred UserCredentials
		if err := c.ShouldBindBodyWithJSON(&cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token, success, err := userService.Login(c.Request.Context(), &cred)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !success {
			c.Status(http.StatusUnauthorized)
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})
}
