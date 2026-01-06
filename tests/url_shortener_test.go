package tests

import (
	"net/http/httptest"
	"os"
	"testing"

	"urlShortener/internal/app"
	"urlShortener/internal/lib/slogdiscard"
	"urlShortener/internal/storage/sqlite"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
)

const (
	testUser     = "test_user"
	testPassword = "test_password"
)

func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	// Create temp database
	tempFile, err := os.CreateTemp("", "test_storage_*.db")
	require.NoError(t, err)
	tempFile.Close()

	storage, err := sqlite.New(tempFile.Name())
	require.NoError(t, err)

	log := slogdiscard.NewDiscardLogger()

	// Use the same router configuration as the real application
	router := app.NewRouter(log, storage, testUser, testPassword)

	server := httptest.NewServer(router)

	cleanup := func() {
		server.Close()
		os.Remove(tempFile.Name())
	}

	return server, cleanup
}

func TestURLShortener_HappyPath(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Generate random test data
	url := gofakeit.URL()
	alias := gofakeit.LetterN(10)

	// Test: Create short URL
	e.POST("/url").
		WithBasicAuth(testUser, testPassword).
		WithJSON(map[string]string{
			"url":   url,
			"alias": alias,
		}).
		Expect().
		Status(200).
		JSON().Object().
		HasValue("status", "OK").
		HasValue("alias", alias)

	// Test: Redirect by alias
	redirectResp := e.GET("/{alias}", alias).
		WithRedirectPolicy(httpexpect.DontFollowRedirects).
		Expect().
		Status(302)

	redirectResp.Header("Location").IsEqual(url)

	// Test: Delete alias
	e.DELETE("/url/{alias}", alias).
		WithBasicAuth(testUser, testPassword).
		Expect().
		Status(200).
		JSON().Object().
		HasValue("status", "OK")

	// Test: Alias no longer exists
	e.GET("/{alias}", alias).
		WithRedirectPolicy(httpexpect.DontFollowRedirects).
		Expect().
		Status(404).
		JSON().Object().
		HasValue("status", "Error").
		HasValue("error", "not found")
}

func TestURLShortener_SaveWithGeneratedAlias(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	url := gofakeit.URL()

	// Create URL without specifying alias
	resp := e.POST("/url").
		WithBasicAuth(testUser, testPassword).
		WithJSON(map[string]string{
			"url": url,
		}).
		Expect().
		Status(200).
		JSON().Object()

	resp.HasValue("status", "OK")

	// Alias should be generated (6 characters)
	alias := resp.Value("alias").String().Raw()
	require.Len(t, alias, 6)

	// Verify redirect works
	e.GET("/{alias}", alias).
		WithRedirectPolicy(httpexpect.DontFollowRedirects).
		Expect().
		Status(302).
		Header("Location").IsEqual(url)
}

func TestURLShortener_DuplicateAlias(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	alias := gofakeit.LetterN(8)

	// Create first URL
	e.POST("/url").
		WithBasicAuth(testUser, testPassword).
		WithJSON(map[string]string{
			"url":   gofakeit.URL(),
			"alias": alias,
		}).
		Expect().
		Status(200)

	// Try to create with same alias
	e.POST("/url").
		WithBasicAuth(testUser, testPassword).
		WithJSON(map[string]string{
			"url":   gofakeit.URL(),
			"alias": alias,
		}).
		Expect().
		Status(409).
		JSON().Object().
		HasValue("status", "Error").
		HasValue("error", "alias already exists")
}

func TestURLShortener_InvalidURL(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Test: Invalid URL format
	e.POST("/url").
		WithBasicAuth(testUser, testPassword).
		WithJSON(map[string]string{
			"url":   "not-a-valid-url",
			"alias": gofakeit.LetterN(6),
		}).
		Expect().
		Status(400).
		JSON().Object().
		HasValue("status", "Error").
		ContainsKey("error")
}

func TestURLShortener_EmptyURL(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Test: Empty URL
	e.POST("/url").
		WithBasicAuth(testUser, testPassword).
		WithJSON(map[string]string{
			"url": "",
		}).
		Expect().
		Status(400).
		JSON().Object().
		HasValue("status", "Error").
		ContainsKey("error")
}

func TestURLShortener_NotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Test: Non-existent alias
	e.GET("/{alias}", "nonexistent").
		WithRedirectPolicy(httpexpect.DontFollowRedirects).
		Expect().
		Status(404).
		JSON().Object().
		HasValue("status", "Error").
		HasValue("error", "not found")
}

func TestURLShortener_DeleteNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Test: Delete non-existent alias
	e.DELETE("/url/{alias}", "nonexistent").
		WithBasicAuth(testUser, testPassword).
		Expect().
		Status(404).
		JSON().Object().
		HasValue("status", "Error").
		HasValue("error", "not found")
}

func TestURLShortener_Unauthorized(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Test: POST without auth
	e.POST("/url").
		WithJSON(map[string]string{
			"url":   gofakeit.URL(),
			"alias": gofakeit.LetterN(6),
		}).
		Expect().
		Status(401)

	// Test: POST with wrong credentials
	e.POST("/url").
		WithBasicAuth("wrong", "credentials").
		WithJSON(map[string]string{
			"url":   gofakeit.URL(),
			"alias": gofakeit.LetterN(6),
		}).
		Expect().
		Status(401)

	// Test: DELETE without auth
	e.DELETE("/url/{alias}", "somealias").
		Expect().
		Status(401)
}

func TestURLShortener_InvalidJSON(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Test: Invalid JSON body
	e.POST("/url").
		WithBasicAuth(testUser, testPassword).
		WithBytes([]byte(`{"url": "https://example.com"`)).
		WithHeader("Content-Type", "application/json").
		Expect().
		Status(400).
		JSON().Object().
		HasValue("status", "Error").
		HasValue("error", "failed to decode request")
}

func TestURLShortener_MultipleURLs(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	e := httpexpect.Default(t, server.URL)

	// Create multiple URLs and verify all work
	urls := make(map[string]string)

	for i := 0; i < 5; i++ {
		url := gofakeit.URL()
		alias := gofakeit.LetterN(8)
		urls[alias] = url

		e.POST("/url").
			WithBasicAuth(testUser, testPassword).
			WithJSON(map[string]string{
				"url":   url,
				"alias": alias,
			}).
			Expect().
			Status(200)
	}

	// Verify all redirects work
	for alias, expectedURL := range urls {
		e.GET("/{alias}", alias).
			WithRedirectPolicy(httpexpect.DontFollowRedirects).
			Expect().
			Status(302).
			Header("Location").IsEqual(expectedURL)
	}

	// Delete all and verify they're gone
	for alias := range urls {
		e.DELETE("/url/{alias}", alias).
			WithBasicAuth(testUser, testPassword).
			Expect().
			Status(200)

		e.GET("/{alias}", alias).
			WithRedirectPolicy(httpexpect.DontFollowRedirects).
			Expect().
			Status(404)
	}
}
