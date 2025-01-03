package handlers

import (
	"api-server/app/middlewares"
	"api-server/domain/models"
	"api-server/domain/services"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func writeError(wsConn *websocket.Conn, err error) error {
	return wsConn.WriteMessage(websocket.BinaryMessage, []byte(fmt.Sprint("{\"error\": \"", err.Error(), "\"}")))
}

func HandleDashboard(tasksService *services.TasksService, jwtCookieAuth *middlewares.JwtCookieAuthenticator) func(*gin.Context) {
	return func(c *gin.Context) {
		userData, err := GetUserFromCtx(c, jwtCookieAuth.AuthCtxKey)
		if err != nil {
			return
		}

		wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		defer wsConn.Close()

		for {
			mt, reqMessage, err := wsConn.ReadMessage()
			if err != nil {
				break
			}

			if mt != websocket.BinaryMessage {
				if err = writeError(wsConn, fmt.Errorf("only binary messages are allowed")); err != nil {
					break
				}
				continue
			}

			var tasksFilter models.TasksFilter
			if err = binding.JSON.BindBody(reqMessage, &tasksFilter); err != nil {
				if err = writeError(wsConn, err); err != nil {
					break
				}
				continue
			}

			tasks, err := tasksService.ListByUserId(c, userData.Id, tasksFilter)
			if err != nil {
				writeError(wsConn, err)
				break
			}

			respMessage, err := json.Marshal(tasks)
			if err != nil {
				writeError(wsConn, err)
				break
			}

			err = wsConn.WriteMessage(websocket.BinaryMessage, respMessage)
			if err != nil {
				break
			}
		}
	}
}
