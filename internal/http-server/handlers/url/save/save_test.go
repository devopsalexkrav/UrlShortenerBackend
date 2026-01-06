package save

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"urlShortener/internal/http-server/handlers/url/save/mocks"
	"urlShortener/internal/lib/slogdiscard"
	"urlShortener/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		mockSetup  func(m *mocks.URLSaver)
		wantCode   int
		wantStatus string
		wantError  string
		wantAlias  string
	}{
		{
			name: "success with custom alias",
			body: `{"url": "https://google.com", "alias": "google"}`,
			mockSetup: func(m *mocks.URLSaver) {
				m.On("SaveURL", "https://google.com", "google").Return(int64(1), nil)
			},
			wantCode:   http.StatusOK,
			wantStatus: "OK",
			wantAlias:  "google",
		},
		{
			name: "success with generated alias",
			body: `{"url": "https://google.com"}`,
			mockSetup: func(m *mocks.URLSaver) {
				m.On("SaveURL", "https://google.com", mock.AnythingOfType("string")).Return(int64(1), nil)
			},
			wantCode:   http.StatusOK,
			wantStatus: "OK",
		},
		{
			name: "alias already exists",
			body: `{"url": "https://google.com", "alias": "google"}`,
			mockSetup: func(m *mocks.URLSaver) {
				m.On("SaveURL", "https://google.com", "google").Return(int64(0), storage.ErrURLExists)
			},
			wantCode:   http.StatusConflict,
			wantStatus: "Error",
			wantError:  "alias already exists",
		},
		{
			name: "save error",
			body: `{"url": "https://google.com", "alias": "google"}`,
			mockSetup: func(m *mocks.URLSaver) {
				m.On("SaveURL", "https://google.com", "google").Return(int64(0), errors.New("unexpected error"))
			},
			wantCode:   http.StatusInternalServerError,
			wantStatus: "Error",
			wantError:  "failed to add url",
		},
		{
			name:       "invalid json",
			body:       `{"url": "https://google.com"`,
			mockSetup:  func(m *mocks.URLSaver) {},
			wantCode:   http.StatusBadRequest,
			wantStatus: "Error",
			wantError:  "failed to decode request",
		},
		{
			name:       "empty url",
			body:       `{"url": ""}`,
			mockSetup:  func(m *mocks.URLSaver) {},
			wantCode:   http.StatusBadRequest,
			wantStatus: "Error",
			wantError:  "field URL is a required field",
		},
		{
			name:       "invalid url format",
			body:       `{"url": "not-a-url"}`,
			mockSetup:  func(m *mocks.URLSaver) {},
			wantCode:   http.StatusBadRequest,
			wantStatus: "Error",
			wantError:  "field URL is not a valid URL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockSaver := mocks.NewURLSaver(t)
			tc.mockSetup(mockSaver)

			handler := New(slogdiscard.NewDiscardLogger(), mockSaver)

			req := httptest.NewRequest(http.MethodPost, "/url", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler(rec, req)

			require.Equal(t, tc.wantCode, rec.Code)

			var resp Response
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tc.wantStatus, resp.Status)

			if tc.wantError != "" {
				assert.Equal(t, tc.wantError, resp.Error)
			}

			if tc.wantAlias != "" {
				assert.Equal(t, tc.wantAlias, resp.Alias)
			}

			if tc.name == "success with generated alias" {
				assert.NotEmpty(t, resp.Alias)
				assert.Len(t, resp.Alias, aliasLength)
			}
		})
	}
}
