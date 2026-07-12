//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBrowserCookieSessionCSRFAndLogout(t *testing.T) {
	ts := startTestServer(t)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	transport := ts.Server.Client().Transport
	client := &http.Client{Transport: transport, Jar: jar}
	rawClient := &http.Client{Transport: transport}

	loginRequest, err := http.NewRequest(
		http.MethodPost,
		ts.Server.URL+"/api/v1/auth/login",
		bytes.NewBufferString(`{"username":"root","password":"`+ts.RootPass+`"}`),
	)
	require.NoError(t, err)
	loginRequest.Host = "localhost"
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse, err := client.Do(loginRequest)
	require.NoError(t, err)
	defer loginResponse.Body.Close()
	loginResponseBody, err := io.ReadAll(loginResponse.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, loginResponse.StatusCode, string(loginResponseBody))

	var loginBody struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(loginResponseBody, &loginBody))
	require.NotEmpty(t, loginBody.Data.Token)

	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range loginResponse.Cookies() {
		switch cookie.Name {
		case "heimdallr_session":
			sessionCookie = cookie
		case "heimdallr_csrf":
			csrfCookie = cookie
		}
	}
	require.NotNil(t, sessionCookie)
	require.True(t, sessionCookie.HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, sessionCookie.SameSite)
	require.Equal(t, "/", sessionCookie.Path)
	require.NotNil(t, csrfCookie)
	require.False(t, csrfCookie.HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, csrfCookie.SameSite)

	safeRequest, err := http.NewRequest(http.MethodGet, ts.Server.URL+"/api/v1/provider?limit=100", nil)
	require.NoError(t, err)
	safeRequest.Host = "localhost"
	safeResponse, err := client.Do(safeRequest)
	require.NoError(t, err)
	require.NoError(t, safeResponse.Body.Close())
	require.Equal(t, http.StatusOK, safeResponse.StatusCode)

	require.Equal(t, http.StatusForbidden, doLogout(t, client, ts.Server.URL, ""))
	require.Equal(t, http.StatusForbidden, doLogout(t, client, ts.Server.URL, "wrong-token"))
	require.Equal(t, http.StatusNoContent, doLogout(t, client, ts.Server.URL, csrfCookie.Value))

	reuseRequest, err := http.NewRequest(http.MethodGet, ts.Server.URL+"/api/v1/provider", nil)
	require.NoError(t, err)
	reuseRequest.Host = "localhost"
	reuseRequest.AddCookie(sessionCookie)
	reuseResponse, err := rawClient.Do(reuseRequest)
	require.NoError(t, err)
	require.NoError(t, reuseResponse.Body.Close())
	require.Equal(t, http.StatusUnauthorized, reuseResponse.StatusCode)

	orphanedRequest, err := http.NewRequest(http.MethodPost, ts.Server.URL+"/api/v1/auth/logout", nil)
	require.NoError(t, err)
	orphanedRequest.Host = "localhost"
	orphanedRequest.AddCookie(&http.Cookie{Name: "heimdallr_session", Value: "expired-session"})
	orphanedResponse, err := rawClient.Do(orphanedRequest)
	require.NoError(t, err)
	require.NoError(t, orphanedResponse.Body.Close())
	require.Equal(t, http.StatusNoContent, orphanedResponse.StatusCode)
	require.Len(t, orphanedResponse.Cookies(), 2)
}

func doLogout(t *testing.T, client *http.Client, baseURL string, csrfToken string) int {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/logout", nil)
	require.NoError(t, err)
	request.Host = "localhost"
	if csrfToken != "" {
		request.Header.Set("X-CSRF-Token", csrfToken)
	}

	response, err := client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	_, err = io.Copy(io.Discard, response.Body)
	require.NoError(t, err)

	return response.StatusCode
}
