package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

func TestSessionCookieMiddlewareSetsSecureCookiesOnLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)

	next := func(_ *gin.Context, _ interface{}) (interface{}, error) {
		return api.Login200JSONResponse{Data: api.LoginResponse{Token: "session-token"}}, nil
	}
	middleware := SessionCookieMiddleware(CookieConfig{
		SessionCookieName: "session",
		CSRFCookieName:    "csrf",
		Secure:            true,
		SessionTTL:        time.Hour,
	})

	_, err := middleware(next, "Login")(ctx, nil)
	require.NoError(t, err)

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 2)
	require.Equal(t, "session", cookies[0].Name)
	require.Equal(t, "session-token", cookies[0].Value)
	require.True(t, cookies[0].HttpOnly)
	require.True(t, cookies[0].Secure)
	require.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
	require.Equal(t, "/", cookies[0].Path)
	require.Equal(t, "csrf", cookies[1].Name)
	require.NotEmpty(t, cookies[1].Value)
	require.False(t, cookies[1].HttpOnly)
	require.True(t, cookies[1].Secure)
	require.Equal(t, http.SameSiteStrictMode, cookies[1].SameSite)
}

func TestSessionCookieMiddlewareClearsCookiesOnLogout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)

	next := func(_ *gin.Context, _ interface{}) (interface{}, error) {
		return api.Logout204Response{}, nil
	}
	middleware := SessionCookieMiddleware(CookieConfig{
		SessionCookieName: "session",
		CSRFCookieName:    "csrf",
	})

	_, err := middleware(next, "Logout")(ctx, nil)
	require.NoError(t, err)

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 2)
	require.Equal(t, -1, cookies[0].MaxAge)
	require.Equal(t, -1, cookies[1].MaxAge)
}
