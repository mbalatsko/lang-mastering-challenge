package services

import (
	"api-server/internal/utils"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenExpiration = time.Hour * 72

var (
	ErrTokenNotValid     = errors.New("token is now valid")
	ErrClaimsParsing     = errors.New("failed to parse JWT claims")
	ErrEmailClaimMissing = errors.New("email field is not present in JWT claims")
)

type JwtTokenProvider struct {
	JwtSecret string
}

func NewJwtTokenProvider() *JwtTokenProvider {
	jwtSecret := utils.MustGetenv("JWT_SECRET")
	return &JwtTokenProvider{JwtSecret: jwtSecret}
}

func (tp *JwtTokenProvider) ProvideWithExp(email string, exp time.Time) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   exp.Unix(),
	})

	tokenString, err := token.SignedString([]byte(tp.JwtSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (tp *JwtTokenProvider) Provide(email string) (string, error) {
	return tp.ProvideWithExp(email, time.Now().Add(TokenExpiration))
}

func (tp *JwtTokenProvider) ParseEmail(tokenString string) (string, error) {
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(tp.JwtSecret), nil
	})
	if err != nil {
		return "", err
	}
	if !parsedToken.Valid {
		return "", ErrTokenNotValid
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrClaimsParsing
	}

	emailI, found := claims["email"]
	if !found {
		return "", ErrEmailClaimMissing
	}

	email, ok := emailI.(string)
	if !ok {
		return "", ErrClaimsParsing
	}
	return email, nil
}
