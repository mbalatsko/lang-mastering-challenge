package middlewares

import (
	"api-server/internal/domain/repos"
	"api-server/internal/domain/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type JwtAuthenticator struct {
	AuthHeader       string
	AuthHeaderPrefix string
	AuthCtxKey       string
	Handler          gin.HandlerFunc
}

func NewJwtAuthenticator(tp *services.JwtTokenProvider, usersRepo *repos.UsersRepo) *JwtAuthenticator {
	const (
		authHeader       = "Authorization"
		authHeaderPrefix = "Bearer"
		authCtxKey       = "User"
	)
	return &JwtAuthenticator{
		AuthHeader:       authHeader,
		AuthHeaderPrefix: authHeaderPrefix,
		AuthCtxKey:       authCtxKey,
		Handler: func(c *gin.Context) {
			headerValue := c.Request.Header.Get(authHeader)
			if headerValue == "" {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			headerParts := strings.Split(headerValue, " ")
			if len(headerParts) != 2 && headerParts[0] != authHeaderPrefix {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			tokenString := headerParts[1]
			email, err := tp.ParseEmail(tokenString)
			if err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			userData, found, err := usersRepo.GetByEmail(c, email)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if !found {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			c.Set(authCtxKey, userData)
			c.Next()
		},
	}
}
