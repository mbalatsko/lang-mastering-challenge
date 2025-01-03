package middlewares

import (
	"api-server/domain/repos"
	"api-server/domain/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type JwtHeaderAuthenticator struct {
	AuthHeader       string
	AuthHeaderPrefix string
	AuthCtxKey       string
	Handler          gin.HandlerFunc
}

type JwtCookieAuthenticator struct {
	AuthCookieKey string
	AuthCtxKey    string
	Handler       gin.HandlerFunc
}

func NewJwtHeaderAuthenticator(tp *services.JwtTokenProvider, usersRepo *repos.UsersRepo) *JwtHeaderAuthenticator {
	const (
		authHeader       = "Authorization"
		authHeaderPrefix = "Bearer"
		authCtxKey       = "User"
	)
	return &JwtHeaderAuthenticator{
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

			userData, err := usersRepo.GetByEmail(c, email)
			if err == repos.ErrNotFound {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.Set(authCtxKey, userData)
			c.Next()
		},
	}
}

func NewJwtCookieAuthenticator(tp *services.JwtTokenProvider, usersRepo *repos.UsersRepo) *JwtCookieAuthenticator {
	const (
		authCookieKey = "auth_token"
		authCtxKey    = "User"
	)
	return &JwtCookieAuthenticator{
		AuthCookieKey: authCookieKey,
		AuthCtxKey:    authCtxKey,
		Handler: func(c *gin.Context) {
			tokenString, err := c.Cookie(authCookieKey)
			if tokenString == "" || err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			email, err := tp.ParseEmail(tokenString)
			if err != nil {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			userData, err := usersRepo.GetByEmail(c, email)
			if err == repos.ErrNotFound {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.Set(authCtxKey, userData)
			c.Next()
		},
	}
}
