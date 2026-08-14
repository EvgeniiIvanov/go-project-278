package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"code/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLinkGeneratedNameRetriesOnCollision(t *testing.T) {
	store := storage.NewFake()
	router := NewRouter(store, testConfig())

	// Occupy the first generated candidate.
	_, err := store.CreateLink(t.Context(), storage.CreateLinkInput{
		OriginalURL: "https://example.com/occupied",
		ShortURL:    "http://localhost:8080/r/collide1",
		ShortName:   "collide1",
	})
	require.NoError(t, err)

	var calls atomic.Int32
	old := generateShortNameFn
	generateShortNameFn = func(length int) (string, error) {
		n := calls.Add(1)
		if n == 1 {
			return "collide1", nil
		}
		return "unique12", nil
	}
	t.Cleanup(func() { generateShortNameFn = old })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(
		`{"original_url":"https://example.com/auto-retry"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unique12", body["short_name"])
	assert.Equal(t, "http://localhost:8080/r/unique12", body["short_url"])
	assert.GreaterOrEqual(t, calls.Load(), int32(2))
}

func TestUpdateLinkValidationAndShortURL(t *testing.T) {
	store := storage.NewFake()
	router := NewRouter(store, testConfig())

	created, err := store.CreateLink(t.Context(), storage.CreateLinkInput{
		OriginalURL: "https://example.com/one",
		ShortURL:    "http://localhost:8080/r/oneone",
		ShortName:   "oneone",
	})
	require.NoError(t, err)
	_, err = store.CreateLink(t.Context(), storage.CreateLinkInput{
		OriginalURL: "https://example.com/two",
		ShortURL:    "http://localhost:8080/r/twotwo",
		ShortName:   "twotwo",
	})
	require.NoError(t, err)

	// invalid JSON
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/links/1", bytes.NewReader([]byte(`{"original_url":`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var invalidJSON map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invalidJSON))
	assert.Equal(t, "invalid request", invalidJSON["error"])

	// invalid url -> 422 field error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/links/1", bytes.NewReader([]byte(
		`{"original_url":"notaurl","short_name":"validname"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var invalidURL map[string]map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invalidURL))
	require.Contains(t, invalidURL["errors"], "original_url")
	assert.Contains(t, invalidURL["errors"]["original_url"], "original_url")
	assert.Contains(t, invalidURL["errors"]["original_url"], "url")

	// short_name too short -> 422
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/links/1", bytes.NewReader([]byte(
		`{"original_url":"https://example.com/x","short_name":"ab"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var shortErr map[string]map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &shortErr))
	require.Contains(t, shortErr["errors"], "short_name")

	// unique conflict -> 422
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/links/1", bytes.NewReader([]byte(
		`{"original_url":"https://example.com/x","short_name":"twotwo"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var conflict map[string]map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &conflict))
	assert.Equal(t, "short name already in use", conflict["errors"]["short_name"])

	// omit short_name -> keep existing, rebuild short_url with /r/
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/links/1", bytes.NewReader([]byte(
		`{"original_url":"https://example.com/updated"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var updated map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, "oneone", updated["short_name"])
	assert.Equal(t, "http://localhost:8080/r/oneone", updated["short_url"])
	assert.Equal(t, "https://example.com/updated", updated["original_url"])

	// unknown id
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/links/999", bytes.NewReader([]byte(
		`{"original_url":"https://example.com/x","short_name":"missing1"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	// custom short_name updates short_url path too
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/links/1", bytes.NewReader([]byte(
		`{"original_url":"https://example.com/final","short_name":"custom12"}`,
	)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var custom map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &custom))
	assert.Equal(t, "custom12", custom["short_name"])
	assert.Equal(t, "http://localhost:8080/r/custom12", custom["short_url"])

	// get/list keep /r/ format
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/links/1", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "http://localhost:8080/r/custom12", got["short_url"])
	assert.EqualValues(t, created.ID, got["id"])
}
