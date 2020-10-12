package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestStatusEndpoint(t *testing.T) {
	req, err := http.NewRequest("GET", "/ansible/status", nil)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	http.HandlerFunc(HandlerStatus).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Received bad status code %s", rr.Code)

	host, err := os.Hostname()
	assert.Nil(t, err)

	expected := string(fmt.Sprintf(`
				{
					"ansible_disabled": true,
					"ansible_last_run_success": true,
					"ansible_running": false,
					"app_name": "ansible-puller",
					"hostname": "%s",
					"version": ""
				}`, host))
	assert.JSONEq(t, expected, rr.Body.String())
}
