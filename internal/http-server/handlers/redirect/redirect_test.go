package redirect

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"urlShortener/internal/http-server/handlers/redirect/mocks"
	resp "urlShortener/internal/lib/api/response"
	"urlShortener/internal/lib/slogdiscard"
	"urlShortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectHandler_EmptyAlias(t *testing.T) {
	mockGetter := mocks.NewURLGetter(t)

	handler := New(slogdiscard.NewDiscardLogger(), mockGetter)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response resp.Response
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request", response.Error)
}

func TestRedirectHandler(t *testing.T) {
	cases := []struct {
		name         string
		alias        string
		mockSetup    func(m *mocks.URLGetter)
		wantRedirect string
		wantStatus   int
		wantError    string
	}{
		{
			name:  "success redirect",
			alias: "google",
			mockSetup: func(m *mocks.URLGetter) {
				m.On("GetURL", "google").Return("https://google.com", nil)
			},
			wantRedirect: "https://google.com",
			wantStatus:   http.StatusFound,
		},
		{
			name:  "url not found",
			alias: "unknown",
			mockSetup: func(m *mocks.URLGetter) {
				m.On("GetURL", "unknown").Return("", storage.ErrURLNotFound)
			},
			wantStatus: http.StatusOK,
			wantError:  "not found",
		},
		{
			name:  "internal error",
			alias: "test",
			mockSetup: func(m *mocks.URLGetter) {
				m.On("GetURL", "test").Return("", errors.New("db error"))
			},
			wantStatus: http.StatusOK,
			wantError:  "internal error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockGetter := mocks.NewURLGetter(t)
			tc.mockSetup(mockGetter)

			handler := New(slogdiscard.NewDiscardLogger(), mockGetter)

			r := chi.NewRouter()
			r.Get("/{alias}", handler)

			req := httptest.NewRequest(http.MethodGet, "/"+tc.alias, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantRedirect != "" {
				assert.Equal(t, tc.wantRedirect, rec.Header().Get("Location"))
			}

			if tc.wantError != "" {
				var response resp.Response
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.wantError, response.Error)
			}
		})
	}
}

