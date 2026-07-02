package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newAuthRouter(t *testing.T) (*gin.Engine, Service, Repository) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	svc, repo, _ := newTestService(t)
	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	api := r.Group("/api")
	RegisterRoutes(api, h, svc)

	return r, svc, repo
}

func createUserFixture(t *testing.T, svc Service, req CreateRequest) {
	t.Helper()

	_, err := svc.Create(t.Context(), req)
	require.NoError(t, err)
}

func doRequest(t *testing.T, router *gin.Engine, method string, path string, body any, username string, password string) *httptest.ResponseRecorder {
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
	if username != "" {
		req.Header.Set(authHeaderUsername, username)
	}
	if password != "" {
		req.Header.Set(authHeaderPassword, password)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

func TestRequireRolesReturnsUnauthorizedWithoutHeaders(t *testing.T) {
	router, _, _ := newAuthRouter(t)

	rr := doRequest(t, router, http.MethodPost, "/api/v1/auth/users", CreateRequest{}, "", "")
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireRolesReturnsUnauthorizedForInvalidCredentials(t *testing.T) {
	router, svc, _ := newAuthRouter(t)

	createUserFixture(t, svc, CreateRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
		Roles:    []string{RoleAdmin},
	})

	rr := doRequest(t, router, http.MethodPost, "/api/v1/auth/users", CreateRequest{}, "admin", "wrong-password")
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireRolesReturnsUnauthorizedForUnknownUser(t *testing.T) {
	router, _, _ := newAuthRouter(t)

	rr := doRequest(t, router, http.MethodPost, "/api/v1/auth/users", CreateRequest{}, "ghost", "AnyPassword123!")
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireRolesReturnsForbiddenForReader(t *testing.T) {
	router, svc, _ := newAuthRouter(t)

	createUserFixture(t, svc, CreateRequest{
		Username: "reader",
		Email:    "reader@example.com",
		Password: "ReaderPassword123!",
		Roles:    []string{RoleReader},
	})

	rr := doRequest(t, router, http.MethodPost, "/api/v1/auth/users", CreateRequest{}, "reader", "ReaderPassword123!")
	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminCanCreateAndDeleteUser(t *testing.T) {
	router, svc, _ := newAuthRouter(t)

	createUserFixture(t, svc, CreateRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
		Roles:    []string{RoleAdmin},
	})

	rr := doRequest(t, router, http.MethodPost, "/api/v1/auth/users", CreateRequest{
		Username: "managed-user",
		Email:    "managed-user@example.com",
		Password: "ManagedPassword123!",
		Roles:    []string{RoleReader},
	}, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusCreated, rr.Code)

	var response struct {
		Data GetResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &response))
	require.NotEmpty(t, response.Data.ID)

	rr = doRequest(t, router, http.MethodDelete, "/api/v1/auth/users/"+response.Data.ID, nil, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestAdminCanListAndUpdateUser(t *testing.T) {
	router, svc, _ := newAuthRouter(t)

	createUserFixture(t, svc, CreateRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
		Roles:    []string{RoleAdmin},
	})

	rr := doRequest(t, router, http.MethodPost, "/api/v1/auth/users", CreateRequest{
		Username: "managed-user",
		Email:    "managed-user@example.com",
		Password: "ManagedPassword123!",
		Roles:    []string{RoleReader},
	}, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusCreated, rr.Code)

	var created struct {
		Data GetResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))

	rr = doRequest(t, router, http.MethodGet, "/api/v1/auth/users", nil, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusOK, rr.Code)

	var listed struct {
		Data []GetResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &listed))
	require.NotEmpty(t, listed.Data)

	rr = doRequest(t, router, http.MethodPut, "/api/v1/auth/users/"+created.Data.ID, UpdateRequest{
		Email: "managed-user-updated@example.com",
		Roles: []string{RoleAdmin},
	}, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusOK, rr.Code)

	var updated struct {
		Data GetResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &updated))
	require.Equal(t, "managed-user-updated@example.com", updated.Data.Email)
	require.Equal(t, []string{RoleAdmin}, updated.Data.Roles)
}

func TestDeleteMapsInvalidAndNotFoundToExpectedStatuses(t *testing.T) {
	router, svc, _ := newAuthRouter(t)

	createUserFixture(t, svc, CreateRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
		Roles:    []string{RoleAdmin},
	})

	rr := doRequest(t, router, http.MethodDelete, "/api/v1/auth/users/%20%20%20", nil, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusBadRequest, rr.Code)

	rr = doRequest(t, router, http.MethodDelete, "/api/v1/auth/users/5d8dd803-fca6-4f7c-9dd2-24417622d630", nil, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteReturnsBadRequestForRootUser(t *testing.T) {
	router, svc, _ := newAuthRouter(t)

	password, err := svc.EnsureRootUser(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, password)

	root, err := svc.Authenticate(t.Context(), rootUsername, password)
	require.NoError(t, err)

	createUserFixture(t, svc, CreateRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
		Roles:    []string{RoleAdmin},
	})

	rr := doRequest(t, router, http.MethodDelete, "/api/v1/auth/users/"+root.ID, nil, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateReturnsBadRequestForRootRoleChange(t *testing.T) {
	router, svc, _ := newAuthRouter(t)

	password, err := svc.EnsureRootUser(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, password)

	root, err := svc.Authenticate(t.Context(), rootUsername, password)
	require.NoError(t, err)

	createUserFixture(t, svc, CreateRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "AdminPassword123!",
		Roles:    []string{RoleAdmin},
	})

	rr := doRequest(t, router, http.MethodPut, "/api/v1/auth/users/"+root.ID, UpdateRequest{
		Roles: []string{RoleReader},
	}, "admin", "AdminPassword123!")
	require.Equal(t, http.StatusBadRequest, rr.Code)
}
