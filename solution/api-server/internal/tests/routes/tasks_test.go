package routes_test

import (
	"api-server/internal/app/middlewares"
	"api-server/internal/app/routes"
	"api-server/internal/db"
	"api-server/internal/domain/models"
	"api-server/internal/domain/repos"
	"api-server/internal/domain/services"
	"api-server/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

var (
	taskNameGen    = rapid.StringMatching(`[^\x00]+`)
	dueDateUnixGen = rapid.Int64Range(time.Now().UTC().Add(-72*time.Hour).Unix(), time.Now().UTC().Add(72*time.Hour).Unix())
	statusGen      = rapid.StringMatching(fmt.Sprintf("^(%s)$", strings.Join(models.ValidTaskStatuses, "|")))
)

func createUserWithTasks(userCred models.UserCredentials, tasks []models.TaskData, userRepo *repos.UsersRepo, tasksRepo *repos.TasksRepo) {
	user, _ := userRepo.Create(context.Background(), userCred.Email, userCred.Password)
	for _, t := range tasks {
		_, err := tasksRepo.CreateWithStatus(context.Background(), t.Name, t.DueDate, t.Status, user.Id)
		if err != nil {
			panic(err)
		}
	}
}

func TestTasksList(t *testing.T) {
	r := gin.Default()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/tasks/", strings.NewReader(""))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Empty list on empty tasks list", func(t *testing.T) {
		defer utils.TruncateTables(conn, []string{"tasks", "users"})
		userCred := models.UserCredentials{Email: "tester@test.com", Password: "whatever"}
		token, _ := tp.Provide(userCred.Email)

		createUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)

		req, _ := http.NewRequest("GET", "/tasks/", strings.NewReader(""))
		req.Header.Set(jwtAuth.AuthHeader, fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token))

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code, resp.Body.String())

		var tasksResp []models.TaskData
		err := json.NewDecoder(resp.Body).Decode(&tasksResp)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, 0, len(tasksResp), tasksResp)
	})

	t.Run("Same list as in DB", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			defer utils.TruncateTables(conn, []string{"tasks", "users"})

			userCred := models.UserCredentials{Email: "tester@test.com", Password: "whatever"}
			token, _ := tp.Provide(userCred.Email)

			tasksCount := rapid.IntRange(1, 10).Draw(t, "tasksCount")

			// create expected tasks
			expectedTasks := make([]models.TaskData, 0, tasksCount)
			for i := 0; i < tasksCount; i++ {
				genDueDate := rapid.Bool().Draw(t, fmt.Sprintf("genDueDate%d", i))
				taskName := taskNameGen.Draw(t, fmt.Sprintf("taskName%d", i))
				status := statusGen.Draw(t, fmt.Sprintf("status%d", i))

				if genDueDate {
					dueDateUnix := dueDateUnixGen.Draw(t, fmt.Sprintf("randomUnixTime%d", i))
					dueDate := time.Unix(dueDateUnix, 0).UTC()
					expectedTasks = append(expectedTasks, models.TaskData{Name: taskName, Status: status, DueDate: &dueDate})
				} else {
					expectedTasks = append(expectedTasks, models.TaskData{Name: taskName, Status: status})
				}
			}
			createUserWithTasks(userCred, expectedTasks, userRepo, tasksRepo)

			// create task of other user
			nowUtc := time.Now().UTC()
			createUserWithTasks(
				models.UserCredentials{Email: "other@email.com", Password: "other"},
				[]models.TaskData{
					{Name: "Other name", DueDate: &nowUtc, Status: "To do"},
					{Name: "Other name2", Status: "In progress"},
				},
				userRepo,
				tasksRepo,
			)

			req, _ := http.NewRequest("GET", "/tasks/", strings.NewReader(""))
			req.Header.Set(jwtAuth.AuthHeader, fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token))

			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, 200, resp.Code, resp.Body.String())

			var tasksResp []models.TaskData
			err := json.NewDecoder(resp.Body).Decode(&tasksResp)
			if err != nil {
				panic(err)
			}

			assert.Equal(t, tasksCount, len(tasksResp), tasksResp)

			for i, task := range tasksResp {
				assert.NotEmpty(t, task.Id)
				assert.NotEmpty(t, task.CreatedAt)
				assert.Equal(t, expectedTasks[i].Name, task.Name)
				assert.Equal(t, expectedTasks[i].DueDate, task.DueDate)
				assert.Empty(t, task.UserId)
			}
		})
	})
}
