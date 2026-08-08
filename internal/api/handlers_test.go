package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code/internal/api"
	"code/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRouter() (*storage.Fake, http.Handler) {
	store := storage.NewFake()
	router := api.NewRouter(store, api.Config{ShortURL: "http://localhost:8080"})
	return store, router
}

func TestCreateListGetUpdateDeleteAndRedirect(t *testing.T) {
	_, router := newTestRouter()

	createBody := map[string]string{
		"original_url": "https://example.com/hello",
		"short_name":   "hello",
	}
	raw, err := json.Marshal(createBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "hello", created["short_name"])
	assert.Equal(t, "http://localhost:8080/hello", created["short_url"])
	assert.Equal(t, "https://example.com/hello", created["original_url"])

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/urls", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/urls/1", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updateBody := map[string]string{
		"original_url": "https://example.org/updated",
		"short_name":   "hello",
	}
	raw, err = json.Marshal(updateBody)
	require.NoError(t, err)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/urls/1", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/hello", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.org/updated", w.Header().Get("Location"))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/urls/1", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/urls/1", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateLinkValidationAndConflicts(t *testing.T) {
	_, router := newTestRouter()

	// invalid url
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader([]byte(`{"original_url":"notaurl","short_name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// missing short_name
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader([]byte(`{"original_url":"https://example.com"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// create once
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader([]byte(`{"original_url":"https://example.com","short_name":"dup"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// conflict
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/urls", bytes.NewReader([]byte(`{"original_url":"https://example.com/2","short_name":"dup"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}
