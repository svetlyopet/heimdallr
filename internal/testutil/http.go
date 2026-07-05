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

func AuthHeaders(username, password string) map[string]string {
	return map[string]string{
		"X-Auth-Username": username,
		"X-Auth-Password": password,
	}
}

func BearerHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}
