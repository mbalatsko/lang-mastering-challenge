package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingRoute(t *testing.T) {
	r := setupDefaultRouter()

	req, _ := http.NewRequest("GET", "/ping", nil)

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Equal(t, "pong", resp.Body.String())
}
