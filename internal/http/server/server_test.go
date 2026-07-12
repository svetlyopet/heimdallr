package server_test

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/ditest"
)

func TestServerHTTPHandlerHealthEndpoint(t *testing.T) {
	injector := ditest.NewServerInjector(t)
	srv := ditest.MustInvokeServer(t, injector)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Host = "localhost"
	rec := httptest.NewRecorder()

	srv.HTTPHandler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"ok"`)
}

func TestServerHasExplicitHTTPResourceLimits(t *testing.T) {
	injector := ditest.NewServerInjector(t)
	srv := ditest.MustInvokeServer(t, injector)
	httpServer := srv.HTTPServer()

	require.Equal(t, 5*time.Second, httpServer.ReadHeaderTimeout)
	require.Equal(t, 15*time.Second, httpServer.ReadTimeout)
	require.Equal(t, 30*time.Second, httpServer.WriteTimeout)
	require.Equal(t, 60*time.Second, httpServer.IdleTimeout)
	require.Equal(t, 1<<20, httpServer.MaxHeaderBytes)
}

func TestServerRejectsOversizedGeneratedJSONBody(t *testing.T) {
	injector := ditest.NewServerInjector(t)
	srv := ditest.MustInvokeServer(t, injector)

	const maxBodyBytes = 5 << 20
	prefix := `{"username":"root","password":"`
	suffix := `"}`
	exactBody := prefix + strings.Repeat("a", maxBodyBytes-len(prefix)-len(suffix)) + suffix

	exactRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(exactBody))
	exactRequest.Host = "localhost"
	exactRequest.Header.Set("Content-Type", "application/json")
	exactResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(exactResponse, exactRequest)
	require.NotEqual(t, http.StatusRequestEntityTooLarge, exactResponse.Code)

	oversizedBody := prefix + strings.Repeat("a", maxBodyBytes-len(prefix)-len(suffix)+1) + suffix
	oversizedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(oversizedBody))
	oversizedRequest.Host = "localhost"
	oversizedRequest.Header.Set("Content-Type", "application/json")
	oversizedResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(oversizedResponse, oversizedRequest)
	require.Equal(t, http.StatusRequestEntityTooLarge, oversizedResponse.Code)

	malformedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":`))
	malformedRequest.Host = "localhost"
	malformedRequest.Header.Set("Content-Type", "application/json")
	malformedResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(malformedResponse, malformedRequest)
	require.Equal(t, http.StatusBadRequest, malformedResponse.Code)
}

func TestServerReadHeaderTimeoutClosesIncompleteRequest(t *testing.T) {
	injector := ditest.NewServerInjector(t)
	srv := ditest.MustInvokeServer(t, injector)
	httpServer := srv.HTTPServer()
	httpServer.ReadHeaderTimeout = 50 * time.Millisecond

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- httpServer.Serve(listener)
	}()
	t.Cleanup(func() {
		require.NoError(t, httpServer.Close())
		require.ErrorIs(t, <-serveDone, http.ErrServerClosed)
	})

	connection, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer connection.Close()
	require.NoError(t, connection.SetDeadline(time.Now().Add(2*time.Second)))
	_, err = connection.Write([]byte("GET /api/health HTTP/1.1\r\nHost: localhost\r\n"))
	require.NoError(t, err)

	startedAt := time.Now()
	buffer := make([]byte, 1)
	_, err = connection.Read(buffer)
	require.True(t, err == nil || !errors.Is(err, os.ErrDeadlineExceeded))
	require.Less(t, time.Since(startedAt), time.Second)
}

func TestServerBrowserSessionCookieCSRFAndLogoutFlow(t *testing.T) {
	injector := ditest.NewServerInjector(t)
	srv := ditest.MustInvokeServer(t, injector)

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"username":"root","password":"IntegrationTestPassword12!"}`),
	)
	loginRequest.Host = "localhost"
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(loginResponse, loginRequest)
	require.Equal(t, http.StatusOK, loginResponse.Code)

	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range loginResponse.Result().Cookies() {
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
	require.NotNil(t, csrfCookie)
	require.False(t, csrfCookie.HttpOnly)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/provider?limit=100", nil)
	getRequest.Host = "localhost"
	getRequest.AddCookie(sessionCookie)
	getResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(getResponse, getRequest)
	require.Equal(t, http.StatusOK, getResponse.Code)

	oversizedPageRequest := httptest.NewRequest(http.MethodGet, "/api/v1/provider?limit=101", nil)
	oversizedPageRequest.Host = "localhost"
	oversizedPageRequest.AddCookie(sessionCookie)
	oversizedPageResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(oversizedPageResponse, oversizedPageRequest)
	require.Equal(t, http.StatusBadRequest, oversizedPageResponse.Code)

	missingCSRFRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	missingCSRFRequest.Host = "localhost"
	missingCSRFRequest.AddCookie(sessionCookie)
	missingCSRFResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(missingCSRFResponse, missingCSRFRequest)
	require.Equal(t, http.StatusForbidden, missingCSRFResponse.Code)

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.Host = "localhost"
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.AddCookie(csrfCookie)
	logoutRequest.Header.Set("X-CSRF-Token", csrfCookie.Value)
	logoutResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(logoutResponse, logoutRequest)
	require.Equal(t, http.StatusNoContent, logoutResponse.Code)
	require.Len(t, logoutResponse.Result().Cookies(), 2)
	require.Equal(t, -1, logoutResponse.Result().Cookies()[0].MaxAge)

	repeatedLogoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	repeatedLogoutRequest.Host = "localhost"
	repeatedLogoutResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(repeatedLogoutResponse, repeatedLogoutRequest)
	require.Equal(t, http.StatusNoContent, repeatedLogoutResponse.Code)

	orphanedLogoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	orphanedLogoutRequest.Host = "localhost"
	orphanedLogoutRequest.AddCookie(&http.Cookie{Name: "heimdallr_session", Value: "expired-session"})
	orphanedLogoutResponse := httptest.NewRecorder()
	srv.HTTPHandler().ServeHTTP(orphanedLogoutResponse, orphanedLogoutRequest)
	require.Equal(t, http.StatusNoContent, orphanedLogoutResponse.Code)
	require.Len(t, orphanedLogoutResponse.Result().Cookies(), 2)
}
