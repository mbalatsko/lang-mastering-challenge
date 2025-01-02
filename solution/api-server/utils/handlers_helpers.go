package utils

import (
	"api-server/domain/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

var ErrGetUserFromCtx = errors.New("failed to get user from context")

func GetUserFromCtx(c *gin.Context, ctxKey string) (models.UserData, error) {
	userDataI, ok := c.Get(ctxKey)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "user is not provided by middleware"})
		return models.UserData{}, ErrGetUserFromCtx
	}

	userData, ok := userDataI.(models.UserData)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "wrong user type provided by middleware"})
		return models.UserData{}, ErrGetUserFromCtx
	}
	return userData, nil
}
