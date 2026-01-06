package delete

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"urlShortener/internal/http-server/handlers/url/delete/mocks"
	resp "urlShortener/internal/lib/api/response"
	"urlShortener/internal/lib/slogdiscard"
	"urlShortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteHandler_EmptyAlias(t *testing.T) {
	mockDeleter := mocks.NewURLDeleter(t)

	handler := New(slogdiscard.NewDiscardLogger(), mockDeleter)

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var response resp.Response
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request", response.Error)
}

func TestDeleteHandler(t *testing.T) {
	cases := []struct {
		name       string
		alias      string
		mockSetup  func(m *mocks.URLDeleter)
		wantStatus int
		wantError  string
	}{
		{
			name:  "success delete",
			alias: "google",
			mockSetup: func(m *mocks.URLDeleter) {
				m.On("DeleteURL", "google").Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "url not found",
			alias: "unknown",
			mockSetup: func(m *mocks.URLDeleter) {
				m.On("DeleteURL", "unknown").Return(storage.ErrURLNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantError:  "not found",
		},
		{
			name:  "internal error",
			alias: "test",
			mockSetup: func(m *mocks.URLDeleter) {
				m.On("DeleteURL", "test").Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "internal error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockDeleter := mocks.NewURLDeleter(t)
			tc.mockSetup(mockDeleter)

			handler := New(slogdiscard.NewDiscardLogger(), mockDeleter)

			r := chi.NewRouter()
			r.Delete("/{alias}", handler)

			req := httptest.NewRequest(http.MethodDelete, "/"+tc.alias, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			require.Equal(t, tc.wantStatus, rec.Code)

			var response resp.Response
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)

			if tc.wantError != "" {
				assert.Equal(t, tc.wantError, response.Error)
			} else {
				assert.Equal(t, "OK", response.Status)
			}
		})
	}
}

