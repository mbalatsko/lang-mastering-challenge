package main

import (
	"api-server/pkg/db"
	"api-server/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

var (
	EmailGen = rapid.StringMatching(`^[a-zA-Z0-9]{1,}@[a-zA-Z0-9]{1,}\.[a-zA-Z]{1,}$`)
)

func generateStrongPassword(t *rapid.T, length int) string {
	const (
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*()-_+=<>?"
		allChars     = lowerChars + upperChars + digitChars + specialChars
	)

	// Ensure at least one of each type of character
	var (
		lowerChar   = lowerChars[rapid.IntRange(0, len(lowerChars)-1).Draw(t, "passwordLowerChar")]
		upperChar   = upperChars[rapid.IntRange(0, len(upperChars)-1).Draw(t, "passwordUpperChar")]
		digitChar   = digitChars[rapid.IntRange(0, len(digitChars)-1).Draw(t, "passwordDigitChar")]
		specialChar = specialChars[rapid.IntRange(0, len(specialChars)-1).Draw(t, "passwordSpecialChar")]
		otherChars  = rapid.StringMatching(fmt.Sprintf("^[%s]{%d}$", string(allChars), length-4)).Draw(t, "passwordOtherChars")
	)

	password := make([]byte, 0, length)
	password = append(password, lowerChar, upperChar, digitChar, specialChar)
	password = append(password, otherChars...)

	rand.Shuffle(len(password), func(i, j int) {
		password[i], password[j] = password[j], password[i]
	})

	return string(password)
}

func truncateTables(conn *pgxpool.Pool, tables []string) {
	batch := &pgx.Batch{}
	for _, t := range tables {
		batch.Queue(fmt.Sprintf("DELETE FROM %s", t))
	}
	err := conn.SendBatch(context.Background(), batch).Close()
	if err != nil {
		panic(err)
	}
}

func TestPingRoute(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest("GET", "/ping", nil)

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Equal(t, "pong", resp.Body.String())
}

func TestRegistration(t *testing.T) {
	router := setupRouter()
	conn := db.ConnectDB()
	userRepo := UserRepository{Db: conn}

	t.Run("Failed on empty body", func(t *testing.T) {
		emptyJson, _ := json.Marshal(map[string]string{})
		req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(string(emptyJson)))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())
	})

	t.Run("Failed on invalid email", func(t *testing.T) {
		invalidEmails := []string{"invalid", "invalid.com", "aaa@aaa", "com.com@"}

		for _, email := range invalidEmails {
			testUser := UserCredentials{
				Email:    email,
				Password: "Password1!",
			}
			userJson, _ := json.Marshal(testUser)
			req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(string(userJson)))
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, 400, resp.Code, email, resp.Body.String())
			assert.Contains(t, resp.Body.String(), "Email")
		}
	})

	t.Run("Failed on invalid password", func(t *testing.T) {
		invalidPasswords := []string{"invalid", "invalid1", "invalid!", ".verylongbutinvalid.", "Invalid"}

		for _, password := range invalidPasswords {
			testUser := UserCredentials{
				Email:    "email@test.com",
				Password: password,
			}
			userJson, _ := json.Marshal(testUser)
			req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(string(userJson)))
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, 400, resp.Code, password, resp.Body.String())
			assert.Contains(t, resp.Body.String(), "Password")
		}
	})

	t.Run("Success", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			defer truncateTables(conn, []string{"users"})

			email := EmailGen.Draw(t, "email")
			password := generateStrongPassword(t, rapid.IntRange(8, 20).Draw(t, "passwordLength"))
			testUser := UserCredentials{
				Email:    email,
				Password: password,
			}
			userJson, _ := json.Marshal(testUser)
			req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(string(userJson)))
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, 201, resp.Code, email, password, resp.Body.String())

			user, found, _ := userRepo.GetByEmail(context.Background(), email)
			assert.True(t, found)
			assert.Equal(t, email, user.Email)
		})
	})
}

func TestLogin(t *testing.T) {
	router := setupRouter()
	conn := db.ConnectDB()

	t.Run("Failed on empty body", func(t *testing.T) {
		emptyJson, _ := json.Marshal(map[string]string{})
		req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(string(emptyJson)))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())
	})

	t.Run("Unauthorized on unregistered credentials", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			email := EmailGen.Draw(t, "email")
			password := generateStrongPassword(t, rapid.IntRange(8, 20).Draw(t, "passwordLength"))
			testUser := UserCredentials{
				Email:    email,
				Password: password,
			}
			userJson, _ := json.Marshal(testUser)
			req, _ := http.NewRequest("POST", "/auth/login", strings.NewReader(string(userJson)))
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, 401, resp.Code, resp.Body.String())
		})
	})

	t.Run("Success after registration", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			defer truncateTables(conn, []string{"users"})

			email := EmailGen.Draw(t, "email")
			password := generateStrongPassword(t, rapid.IntRange(8, 20).Draw(t, "passwordLength"))
			testUser := UserCredentials{
				Email:    email,
				Password: password,
			}
			userJson, _ := json.Marshal(testUser)

			req, _ := http.NewRequest("POST", "/auth/register", strings.NewReader(string(userJson)))
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, 201, resp.Code, resp.Body.String())

			req, _ = http.NewRequest("POST", "/auth/login", strings.NewReader(string(userJson)))
			resp = httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, 200, resp.Code, resp.Body.String())

			respMap := map[string]string{}

			err := json.Unmarshal(resp.Body.Bytes(), &respMap)
			assert.Nil(t, err, resp.Body.String())

			tokenString := respMap["token"]
			parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(utils.MustGetenv("JWT_SECRET")), nil
			})
			assert.Nil(t, err, resp.Body.String(), tokenString)
			assert.True(t, parsedToken.Valid)

			claims, _ := parsedToken.Claims.(jwt.MapClaims)
			assert.Equal(t, testUser.Email, claims["email"])
		})
	})
}
