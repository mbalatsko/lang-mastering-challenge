package routes_test

import (
	"api-server/app/middlewares"
	"api-server/app/routes"
	"api-server/db"
	"api-server/domain/models"
	"api-server/domain/repos"
	"api-server/domain/services"
	"api-server/utils"
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

func createUserWithTasks(
	userCred models.UserRegister,
	tasksData []models.TaskData,
	userRepo *repos.UsersRepo,
	tasksRepo *repos.TasksRepo,
) (models.UserData, []models.TaskData) {
	createdTasks := make([]models.TaskData, 0, len(tasksData))
	user, _ := userRepo.Create(context.Background(), userCred.Email, userCred.Password)
	for _, t := range tasksData {
		createdTask, err := tasksRepo.CreateWithStatus(context.Background(), t.Name, t.DueDate, t.Status, user.Id)
		if err != nil {
			panic(err)
		}
		createdTasks = append(createdTasks, createdTask)
	}
	return user, createdTasks
}

func genTask(t *rapid.T, i int) models.TaskData {
	genDueDate := rapid.Bool().Draw(t, fmt.Sprintf("genDueDate%d", i))
	taskName := taskNameGen.Draw(t, fmt.Sprintf("taskName%d", i))
	status := statusGen.Draw(t, fmt.Sprintf("status%d", i))

	if genDueDate {
		dueDateUnix := dueDateUnixGen.Draw(t, fmt.Sprintf("randomUnixTime%d", i))
		dueDate := time.Unix(dueDateUnix, 0).UTC()
		return models.TaskData{Name: taskName, Status: status, DueDate: &dueDate}
	}
	return models.TaskData{Name: taskName, Status: status}
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
		userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
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

			userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
			token, _ := tp.Provide(userCred.Email)

			tasksCount := rapid.IntRange(1, 10).Draw(t, "tasksCount")

			// create expected tasks
			expectedTasks := make([]models.TaskData, 0, tasksCount)
			for i := 0; i < tasksCount; i++ {
				task := genTask(t, i)
				expectedTasks = append(expectedTasks, task)
			}
			createUserWithTasks(userCred, expectedTasks, userRepo, tasksRepo)

			// create task of other user
			nowUtc := time.Now().UTC()
			createUserWithTasks(
				models.UserRegister{Email: "other@email.com", Password: "other"},
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

func TestTasksCreate(t *testing.T) {
	r := gin.Default()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	// create tester user
	userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
	token, _ := tp.Provide(userCred.Email)
	userAuthHeader := fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token)

	createUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)
	defer utils.TruncateTables(conn, []string{"users"})

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/tasks/", strings.NewReader(""))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Failed on invalid body", func(t *testing.T) {
		defer utils.TruncateTables(conn, []string{"tasks"})

		// empty body
		req, _ := http.NewRequest("POST", "/tasks/", strings.NewReader(""))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())

		// task name empty
		testTask := models.TaskCreate{
			Name: "",
		}
		userJson, _ := json.Marshal(testTask)
		req, _ = http.NewRequest("POST", "/tasks/", strings.NewReader(string(userJson)))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp = httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())
	})

	t.Run("Success", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			defer utils.TruncateTables(conn, []string{"tasks"})

			task := genTask(t, 0)

			taskCreate := models.TaskCreate{
				Name:    task.Name,
				DueDate: task.DueDate,
			}

			taskJson, _ := json.Marshal(taskCreate)
			req, _ := http.NewRequest("POST", "/tasks/", strings.NewReader(string(taskJson)))
			req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, 200, resp.Code, resp.Body.String())

			var taskDataResp models.TaskData
			err := json.Unmarshal(resp.Body.Bytes(), &taskDataResp)
			assert.Nil(t, err, resp.Body.String())
			assert.Equal(t, taskDataResp.Name, taskCreate.Name)
			assert.Equal(t, taskDataResp.DueDate, taskCreate.DueDate)
			assert.Equal(t, taskDataResp.Status, "To do")

			taskDataDb, found, err := tasksRepo.GetById(context.Background(), taskDataResp.Id)
			if err != nil {
				panic(err)
			}
			assert.True(t, found, taskDataResp)
			assert.Equal(t, taskDataDb.Name, taskDataResp.Name)
			assert.Equal(t, taskDataDb.DueDate, taskDataResp.DueDate)
			assert.Equal(t, taskDataDb.CreatedAt, taskDataResp.CreatedAt)
			assert.Equal(t, taskDataDb.Status, taskDataResp.Status)
		})
	})
}

func TestTasksDelete(t *testing.T) {
	r := gin.Default()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	// create tester user
	userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
	token, _ := tp.Provide(userCred.Email)
	userAuthHeader := fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token)

	userData, _ := createUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)
	defer utils.TruncateTables(conn, []string{"users"})

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/tasks/123", strings.NewReader(""))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Bad request on non number id", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/tasks/abcd", strings.NewReader(""))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())
	})

	t.Run("Not found on non existing task", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/tasks/123", strings.NewReader(""))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 404, resp.Code, resp.Body.String())
	})

	t.Run("Forbidden on someone else's task", func(t *testing.T) {
		otherUserCred := models.UserRegister{Email: "other@other.com", Password: "whatever"}
		_, createdTasks := createUserWithTasks(otherUserCred, []models.TaskData{{Name: "Other task", Status: "Done"}}, userRepo, tasksRepo)
		createdTask := createdTasks[0]

		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/tasks/%d", createdTask.Id), strings.NewReader(""))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 403, resp.Code, resp.Body.String())
	})

	t.Run("Success", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			defer utils.TruncateTables(conn, []string{"tasks"})

			taskData := genTask(t, 0)
			createdTask, err := tasksRepo.CreateWithStatus(context.Background(), taskData.Name, taskData.DueDate, taskData.Status, userData.Id)
			if err != nil {
				panic(err)
			}

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/tasks/%d", createdTask.Id), strings.NewReader(""))
			req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, 204, resp.Code, resp.Body.String())

			_, found, _ := tasksRepo.GetById(context.Background(), createdTask.Id)
			assert.False(t, found, resp.Code, resp.Body.String())
		})
	})
}

func TestTasksUpdate(t *testing.T) {
	r := gin.Default()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	// create tester user
	userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
	token, _ := tp.Provide(userCred.Email)
	userAuthHeader := fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token)

	userData, _ := createUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)
	defer utils.TruncateTables(conn, []string{"users"})

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("PATCH", "/tasks/123", strings.NewReader(""))
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Bad request on non number id", func(t *testing.T) {
		req, _ := http.NewRequest("PATCH", "/tasks/abcd", strings.NewReader(""))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())
	})

	t.Run("Not found on non existing task", func(t *testing.T) {
		statusUpdate := models.TaskStatus{Status: "To do"}
		statusUpdateJson, _ := json.Marshal(statusUpdate)
		req, _ := http.NewRequest("PATCH", "/tasks/123", strings.NewReader(string(statusUpdateJson)))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 404, resp.Code, resp.Body.String())
	})

	t.Run("Forbidden on someone else's task", func(t *testing.T) {
		otherUserCred := models.UserRegister{Email: "other@other.com", Password: "whatever"}
		_, createdTasks := createUserWithTasks(otherUserCred, []models.TaskData{{Name: "Other task", Status: "Done"}}, userRepo, tasksRepo)
		createdTask := createdTasks[0]

		statusUpdate := models.TaskStatus{Status: "To do"}
		statusUpdateJson, _ := json.Marshal(statusUpdate)
		req, _ := http.NewRequest("PATCH", fmt.Sprintf("/tasks/%d", createdTask.Id), strings.NewReader(string(statusUpdateJson)))
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 403, resp.Code, resp.Body.String())
	})

	t.Run("Success", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			defer utils.TruncateTables(conn, []string{"tasks"})

			taskData := genTask(t, 0)
			createdTask, err := tasksRepo.CreateWithStatus(context.Background(), taskData.Name, taskData.DueDate, taskData.Status, userData.Id)
			if err != nil {
				panic(err)
			}

			statusUpdate := models.TaskStatus{Status: statusGen.Draw(t, "newStatus")}
			statusUpdateJson, _ := json.Marshal(statusUpdate)
			req, _ := http.NewRequest("PATCH", fmt.Sprintf("/tasks/%d", createdTask.Id), strings.NewReader(string(statusUpdateJson)))
			req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, 200, resp.Code, resp.Body.String())

			var updatedTaskData models.TaskData
			err = json.Unmarshal(resp.Body.Bytes(), &updatedTaskData)
			assert.Nil(t, err, resp.Body.String())
			assert.Equal(t, updatedTaskData.Name, createdTask.Name)
			assert.Equal(t, updatedTaskData.DueDate, createdTask.DueDate)
			// status changed
			assert.Equal(t, updatedTaskData.Status, statusUpdate.Status)

			taskDataDb, found, err := tasksRepo.GetById(context.Background(), createdTask.Id)
			if err != nil {
				panic(err)
			}
			assert.True(t, found)
			assert.Equal(t, taskDataDb.Name, createdTask.Name)
			assert.Equal(t, taskDataDb.DueDate, createdTask.DueDate)
			assert.Equal(t, taskDataDb.CreatedAt, createdTask.CreatedAt)
			assert.Equal(t, taskDataDb.Status, statusUpdate.Status)
		})
	})
}
