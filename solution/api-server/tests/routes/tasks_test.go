package routes

import (
	"api-server/app/middlewares"
	"api-server/app/routes"
	"api-server/db"
	"api-server/domain/models"
	"api-server/domain/repos"
	"api-server/domain/services"
	test_utils "api-server/tests/utils"
	"api-server/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

var (
	taskNameGen    = rapid.StringMatching(`[^\x00]+`)
	dueDateUnixGen = rapid.Int64Range(time.Now().UTC().Add(-72*time.Hour).Unix(), time.Now().UTC().Add(72*time.Hour).Unix())
	statusGen      = rapid.StringMatching(fmt.Sprintf("^(%s)$", strings.Join(utils.ValidTaskStatuses, "|")))
)

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
	r := routes.SetupDefaultRouter()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtHeaderAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/tasks/", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Empty list on empty tasks list", func(t *testing.T) {
		defer utils.TruncateTables(conn, []string{"tasks", "users"})
		userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
		token, _ := tp.Provide(userCred.Email)

		test_utils.CreateUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)

		req, _ := http.NewRequest("GET", "/tasks/", nil)
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

	t.Run("Bad request on invalid filters", func(t *testing.T) {
		defer utils.TruncateTables(conn, []string{"tasks", "users"})
		userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
		token, _ := tp.Provide(userCred.Email)

		test_utils.CreateUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)

		u, _ := url.Parse("/tasks/")
		query := u.Query()
		// task due date filter
		query.Set("due_date", "random")

		u.RawQuery = query.Encode()
		req, _ := http.NewRequest("GET", u.String(), nil)
		req.Header.Set(jwtAuth.AuthHeader, fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token))

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())

		// task status filter
		query = u.Query()
		query.Set("status", "invalid")
		u.RawQuery = query.Encode()
		req, _ = http.NewRequest("GET", u.String(), nil)
		req.Header.Set(jwtAuth.AuthHeader, fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token))

		resp = httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())
	})

	t.Run("Successful filtering", func(t *testing.T) {
		defer utils.TruncateTables(conn, []string{"tasks", "users"})
		userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
		token, _ := tp.Provide(userCred.Email)

		timeNow := time.Now().UTC()
		_, tasks := test_utils.CreateUserWithTasks(
			userCred,
			[]models.TaskData{
				{Name: "Task 1", DueDate: &timeNow, Status: "To do"},
				{Name: "Task 2", DueDate: &timeNow, Status: "In progress"},
				{Name: "Task 3", DueDate: nil, Status: "To do"},
				{Name: "Another 3", DueDate: &timeNow, Status: "Done"},
			},
			userRepo,
			tasksRepo,
		)

		u, _ := url.Parse("/tasks/")
		query := u.Query()
		// task name filter
		query.Set("q", "Task")
		u.RawQuery = query.Encode()

		req, _ := http.NewRequest("GET", u.String(), nil)
		req.Header.Set(jwtAuth.AuthHeader, fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token))

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code, resp.Body.String())

		var tasksResp []models.TaskData
		err := json.NewDecoder(resp.Body).Decode(&tasksResp)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, 3, len(tasksResp), tasksResp)
		assert.ElementsMatch(t, test_utils.MapTasksToName(tasks[:3]), test_utils.MapTasksToName(tasksResp))

		// task due date filter
		query.Set("due_date", timeNow.Format(utils.DayDateFmt))
		u.RawQuery = query.Encode()
		req, _ = http.NewRequest("GET", u.String(), nil)
		req.Header.Set(jwtAuth.AuthHeader, fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token))

		resp = httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code, resp.Body.String())

		err = json.NewDecoder(resp.Body).Decode(&tasksResp)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, 2, len(tasksResp), tasksResp)
		assert.ElementsMatch(t, test_utils.MapTasksToName(tasks[:2]), test_utils.MapTasksToName(tasksResp))

		// task status filter
		query.Set("status", "To do")
		u.RawQuery = query.Encode()
		req, _ = http.NewRequest("GET", u.String(), nil)
		req.Header.Set(jwtAuth.AuthHeader, fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token))

		resp = httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code, resp.Body.String())

		err = json.NewDecoder(resp.Body).Decode(&tasksResp)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, 1, len(tasksResp), tasksResp)
		assert.ElementsMatch(t, test_utils.MapTasksToName(tasks[:1]), test_utils.MapTasksToName(tasksResp))
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
			test_utils.CreateUserWithTasks(userCred, expectedTasks, userRepo, tasksRepo)

			// create task of other user
			nowUtc := time.Now().UTC()
			test_utils.CreateUserWithTasks(
				models.UserRegister{Email: "other@email.com", Password: "other"},
				[]models.TaskData{
					{Name: "Other name", DueDate: &nowUtc, Status: "To do"},
					{Name: "Other name2", Status: "In progress"},
				},
				userRepo,
				tasksRepo,
			)

			req, _ := http.NewRequest("GET", "/tasks/", nil)
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
	r := routes.SetupDefaultRouter()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtHeaderAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	// create tester user
	userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
	token, _ := tp.Provide(userCred.Email)
	userAuthHeader := fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token)

	test_utils.CreateUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)
	defer utils.TruncateTables(conn, []string{"users"})

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/tasks/", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Failed on invalid body", func(t *testing.T) {
		defer utils.TruncateTables(conn, []string{"tasks"})

		// empty body
		req, _ := http.NewRequest("POST", "/tasks/", nil)
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

			taskDataDb, err := tasksRepo.GetById(context.Background(), taskDataResp.Id)
			assert.Nil(t, err)
			assert.Equal(t, taskDataDb.Name, taskDataResp.Name)
			assert.Equal(t, taskDataDb.DueDate, taskDataResp.DueDate)
			assert.Equal(t, taskDataDb.CreatedAt, taskDataResp.CreatedAt)
			assert.Equal(t, taskDataDb.Status, taskDataResp.Status)
		})
	})
}

func TestTasksDelete(t *testing.T) {
	r := routes.SetupDefaultRouter()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtHeaderAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	// create tester user
	userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
	token, _ := tp.Provide(userCred.Email)
	userAuthHeader := fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token)

	userData, _ := test_utils.CreateUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)
	defer utils.TruncateTables(conn, []string{"users"})

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/tasks/123", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Bad request on non number id", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/tasks/abcd", nil)
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code, resp.Body.String())
	})

	t.Run("Not found on non existing task", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/tasks/123", nil)
		req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 404, resp.Code, resp.Body.String())
	})

	t.Run("Forbidden on someone else's task", func(t *testing.T) {
		otherUserCred := models.UserRegister{Email: "other@other.com", Password: "whatever"}
		_, createdTasks := test_utils.CreateUserWithTasks(otherUserCred, []models.TaskData{{Name: "Other task", Status: "Done"}}, userRepo, tasksRepo)
		createdTask := createdTasks[0]

		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/tasks/%d", createdTask.Id), nil)
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

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/tasks/%d", createdTask.Id), nil)
			req.Header.Set(jwtAuth.AuthHeader, userAuthHeader)

			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, 204, resp.Code, resp.Body.String())

			_, err = tasksRepo.GetById(context.Background(), createdTask.Id)
			assert.Equal(t, repos.ErrNotFound, err, resp.Code)
		})
	})
}

func TestTasksUpdate(t *testing.T) {
	r := routes.SetupDefaultRouter()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtAuth := middlewares.NewJwtHeaderAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterTasksRoutes(r, jwtAuth, tasksService)

	// create tester user
	userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
	token, _ := tp.Provide(userCred.Email)
	userAuthHeader := fmt.Sprintf("%s %s", jwtAuth.AuthHeaderPrefix, token)

	userData, _ := test_utils.CreateUserWithTasks(userCred, []models.TaskData{}, userRepo, tasksRepo)
	defer utils.TruncateTables(conn, []string{"users"})

	t.Run("Unauthorized on empty header", func(t *testing.T) {
		req, _ := http.NewRequest("PATCH", "/tasks/123", nil)
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code, resp.Body.String())
	})

	t.Run("Bad request on non number id", func(t *testing.T) {
		req, _ := http.NewRequest("PATCH", "/tasks/abcd", nil)
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
		_, createdTasks := test_utils.CreateUserWithTasks(otherUserCred, []models.TaskData{{Name: "Other task", Status: "Done"}}, userRepo, tasksRepo)
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

			taskDataDb, err := tasksRepo.GetById(context.Background(), createdTask.Id)
			assert.Nil(t, err)
			assert.Equal(t, taskDataDb.Name, createdTask.Name)
			assert.Equal(t, taskDataDb.DueDate, createdTask.DueDate)
			assert.Equal(t, taskDataDb.CreatedAt, createdTask.CreatedAt)
			assert.Equal(t, taskDataDb.Status, statusUpdate.Status)
		})
	})
}
