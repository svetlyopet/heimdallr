package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func DoJSONRequest(t *testing.T, router http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		payload = encoded
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Host = "localhost"

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

func DoGinJSONRequest(t *testing.T, router *gin.Engine, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	return DoJSONRequest(t, router, method, path, body, headers)
}

func AssertJSONStatus(t *testing.T, rr *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()

	require.Equal(t, status, rr.Code)

	var response map[string]any
	if rr.Body.Len() > 0 {
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	}

	return response
}

func BearerHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

func LoginBearerHeader(t *testing.T, router http.Handler, username, password string) map[string]string {
	t.Helper()

	rr := DoJSONRequest(t, router, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, nil)
	require.Equal(t, http.StatusOK, rr.Code)

	var response struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.NotEmpty(t, response.Data.Token)

	return BearerHeader(response.Data.Token)
}
