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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestDashboard(t *testing.T) {
	r := routes.SetupDefaultRouter()

	conn := db.ConnectDB()

	tp := services.NewJwtTokenProvider()
	userRepo := repos.NewUsersRepo(conn)

	tasksRepo := repos.NewTasksRepo(conn)
	tasksService := services.NewTasksService(tasksRepo)

	jwtCookieAuth := middlewares.NewJwtCookieAuthenticator(tp, userRepo)

	utils.RegisterValidators()
	routes.RegisterDashboardRoute(r, jwtCookieAuth, tasksService)

	// Start a test server
	server := httptest.NewServer(r)
	defer server.Close()

	u := &url.URL{
		Scheme: "ws",
		Host:   server.URL[7:],
		Path:   "/dashboard/",
	}

	// create tester user
	userCred := models.UserRegister{Email: "tester@test.com", Password: "whatever"}
	token, _ := tp.Provide(userCred.Email)
	header := http.Header{"Cookie": {fmt.Sprintf("%s=%s", jwtCookieAuth.AuthCookieKey, token)}}

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

	defer utils.TruncateTables(conn, []string{"tasks", "users"})

	t.Run("Bad handshake on empty cookies", func(t *testing.T) {
		_, httpResp, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.EqualError(t, err, websocket.ErrBadHandshake.Error())
		assert.Equal(t, http.StatusUnauthorized, httpResp.StatusCode)
	})

	t.Run("Successful connection with auth cookie", func(t *testing.T) {
		header := http.Header{"Cookie": {fmt.Sprintf("%s=%s", jwtCookieAuth.AuthCookieKey, token)}}
		_, _, err := websocket.DefaultDialer.Dial(u.String(), header)
		assert.NoError(t, err)
	})

	t.Run("Error on invalid filters", func(t *testing.T) {
		wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
		assert.NoError(t, err)
		defer wsConn.Close()

		// non binary message
		testMessage := "Hello, WebSocket!"
		err = wsConn.WriteMessage(websocket.BinaryMessage, []byte(testMessage))
		assert.NoError(t, err)
		_, message, err := wsConn.ReadMessage()
		assert.NoError(t, err)
		assert.Contains(t, string(message), "error")

		// random due date
		randomDueDate := "random"
		tasksFilter := models.TasksFilter{DueDateStr: &randomDueDate}
		reqBytes, _ := json.Marshal(tasksFilter)
		err = wsConn.WriteMessage(websocket.BinaryMessage, reqBytes)
		assert.NoError(t, err)
		_, message, err = wsConn.ReadMessage()
		assert.NoError(t, err)
		assert.Contains(t, string(message), "error")

		// random status
		randomStatus := "random"
		tasksFilter = models.TasksFilter{Status: &randomStatus}
		reqBytes, _ = json.Marshal(tasksFilter)
		err = wsConn.WriteMessage(websocket.BinaryMessage, reqBytes)
		assert.NoError(t, err)
		_, message, err = wsConn.ReadMessage()
		assert.NoError(t, err)
		assert.Contains(t, string(message), "error")
	})

	t.Run("Success", func(t *testing.T) {
		wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
		assert.NoError(t, err)
		defer wsConn.Close()

		// empty filter, all tasks
		tasksFilter := models.TasksFilter{}
		reqBytes, _ := json.Marshal(tasksFilter)
		err = wsConn.WriteMessage(websocket.BinaryMessage, reqBytes)
		assert.NoError(t, err)
		_, resp, err := wsConn.ReadMessage()
		assert.NoError(t, err)
		var tasksResp []models.TaskData
		err = json.Unmarshal(resp, &tasksResp)
		assert.NoError(t, err, string(resp))
		assert.Equal(t, 4, len(tasksResp), tasksResp)
		assert.ElementsMatch(t, test_utils.MapTasksToName(tasks), test_utils.MapTasksToName(tasksResp))

		// query
		query := "Task"

		tasksFilter = models.TasksFilter{Query: &query}
		reqBytes, _ = json.Marshal(tasksFilter)
		err = wsConn.WriteMessage(websocket.BinaryMessage, reqBytes)
		assert.NoError(t, err)
		_, resp, err = wsConn.ReadMessage()
		assert.NoError(t, err)
		err = json.Unmarshal(resp, &tasksResp)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(tasksResp), tasksResp)
		assert.ElementsMatch(t, test_utils.MapTasksToName(tasks[:3]), test_utils.MapTasksToName(tasksResp))

		// query + due date
		dueDate := timeNow.Format(utils.DayDateFmt)

		tasksFilter = models.TasksFilter{Query: &query, DueDateStr: &dueDate}
		reqBytes, _ = json.Marshal(tasksFilter)
		err = wsConn.WriteMessage(websocket.BinaryMessage, reqBytes)
		assert.NoError(t, err)
		_, resp, err = wsConn.ReadMessage()
		assert.NoError(t, err)
		err = json.Unmarshal(resp, &tasksResp)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(tasksResp), tasksResp)
		assert.ElementsMatch(t, test_utils.MapTasksToName(tasks[:2]), test_utils.MapTasksToName(tasksResp))

		// query + due date + status
		status := "To do"

		tasksFilter = models.TasksFilter{Query: &query, DueDateStr: &dueDate, Status: &status}
		reqBytes, _ = json.Marshal(tasksFilter)
		err = wsConn.WriteMessage(websocket.BinaryMessage, reqBytes)
		assert.NoError(t, err)
		_, resp, err = wsConn.ReadMessage()
		assert.NoError(t, err)
		err = json.Unmarshal(resp, &tasksResp)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(tasksResp), tasksResp)
		assert.ElementsMatch(t, test_utils.MapTasksToName(tasks[:1]), test_utils.MapTasksToName(tasksResp))
	})
}
