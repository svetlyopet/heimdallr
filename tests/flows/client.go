package flows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Client performs HTTP requests against a Heimdallr API instance.
type Client struct {
	t           *testing.T
	baseURL     string
	httpClient  *http.Client
	authHeaders map[string]string
}

// NewLiveClient connects to a running Heimdallr instance using environment variables.
func NewLiveClient(t *testing.T) *Client {
	t.Helper()

	baseURL := envOrDefault("HEIMDALLR_URL", "http://localhost:8080")
	user := envOrDefault("HEIMDALLR_USER", "root")
	password := envOrDefault("HEIMDALLR_PASSWORD", "e2e-test-password")

	token := LoginToken(t, baseURL, http.DefaultClient, user, password)
	return NewClient(t, baseURL, http.DefaultClient, token)
}

// NewClient creates a client with the given base URL and bearer token.
func NewClient(t *testing.T, baseURL string, httpClient *http.Client, token string) *Client {
	t.Helper()

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		t:           t,
		baseURL:     baseURL,
		httpClient:  httpClient,
		authHeaders: BearerHeader(token),
	}
}

// WithToken returns a copy of the client using a different bearer token.
func (c *Client) WithToken(token string) *Client {
	return &Client{
		t:           c.t,
		baseURL:     c.baseURL,
		httpClient:  c.httpClient,
		authHeaders: BearerHeader(token),
	}
}

// BaseURL returns the API base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Request performs an HTTP request and parses the JSON response body.
func (c *Client) Request(method, path string, body any, headers map[string]string) (*http.Response, map[string]any) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	require.NoError(c.t, err)
	req.Host = "localhost"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range c.authHeaders {
		req.Header.Set(key, value)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	require.NoError(c.t, err)

	var parsed map[string]any
	if resp.Body != nil {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		require.NoError(c.t, readErr)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		if len(bodyBytes) > 0 {
			require.NoError(c.t, json.Unmarshal(bodyBytes, &parsed))
		}
	}

	return resp, parsed
}

// LoginToken authenticates and returns a bearer token.
func LoginToken(t *testing.T, baseURL string, httpClient *http.Client, username, password string) string {
	t.Helper()

	client := NewClient(t, baseURL, httpClient, "")
	resp, parsed := client.Request(http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	data := DataField(t, parsed)
	token, ok := data["token"].(string)
	require.True(t, ok, fmt.Sprintf("expected login token, got %#v", parsed))
	return token
}

// BearerHeader returns Authorization headers for the given token.
func BearerHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

// DataField extracts the "data" object from an API response.
func DataField(t *testing.T, parsed map[string]any) map[string]any {
	t.Helper()

	data, ok := parsed["data"].(map[string]any)
	require.True(t, ok, fmt.Sprintf("expected data object, got %#v", parsed))
	return data
}

// ListDataField extracts the "data" array from an API response.
func ListDataField(t *testing.T, parsed map[string]any) []any {
	t.Helper()

	data, ok := parsed["data"].([]any)
	require.True(t, ok, fmt.Sprintf("expected data array, got %#v", parsed))
	return data
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
