package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
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
	router := api.NewRouter(store, testConfig())
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
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "hello", created["short_name"])
	assert.Equal(t, "http://localhost:8080/r/hello", created["short_url"])
	assert.Equal(t, "https://example.com/hello", created["original_url"])
	assert.Contains(t, created["short_url"], "/r/")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/links?range=[0,10]", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "links 0-0/1", w.Header().Get("Content-Range"))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/links/1", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updateBody := map[string]string{
		"original_url": "https://example.org/updated",
		"short_name":   "hello",
	}
	raw, err = json.Marshal(updateBody)
	require.NoError(t, err)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/links/1", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, "https://example.org/updated", updated["original_url"])
	assert.Equal(t, "hello", updated["short_name"])

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/r/hello", nil)
	req.Header.Set("User-Agent", "curl/8.5.0")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.org/updated", w.Header().Get("Location"))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/links/1", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/links/1", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateLinkValidationAndConflicts(t *testing.T) {
	_, router := newTestRouter()

	// invalid JSON
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(`{"original_url":`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var invalidJSON map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &invalidJSON))
	assert.Equal(t, "invalid request", invalidJSON["error"])

	// invalid url -> 422 field error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(`{"original_url":"notaurl","short_name":"demo"}`)))
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
	req = httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(`{"original_url":"https://example.com","short_name":"ab"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var shortNameErr map[string]map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &shortNameErr))
	require.Contains(t, shortNameErr["errors"], "short_name")

	// missing short_name -> auto-generated 8-char name
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(`{"original_url":"https://example.com/auto"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var autoCreated map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &autoCreated))
	autoName, ok := autoCreated["short_name"].(string)
	require.True(t, ok)
	assert.Len(t, autoName, 8)
	assert.Equal(t, "http://localhost:8080/r/"+autoName, autoCreated["short_url"])

	// create once
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(`{"original_url":"https://example.com","short_name":"dup"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// unique conflict -> 422 field error on short_name
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte(`{"original_url":"https://example.com/2","short_name":"dup"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var conflict map[string]map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &conflict))
	assert.Equal(t, "short name already in use", conflict["errors"]["short_name"])
}

func TestListLinksPagination(t *testing.T) {
	store, router := newTestRouter()

	names := []string{"a", "b", "c"}
	for _, name := range names {
		_, err := store.CreateLink(t.Context(), storage.CreateLinkInput{
			OriginalURL: "https://example.com/" + name,
			ShortURL:    "http://localhost:8080/" + name,
			ShortName:   name,
		})
		require.NoError(t, err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/links?range=[1,2]", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "links 1-2/3", w.Header().Get("Content-Range"))

	var page []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	require.Len(t, page, 2)

	// Hexlet-style large page with space after comma.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/links?range=%5B0,+1000%5D", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "links 0-2/3", w.Header().Get("Content-Range"))
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	require.Len(t, page, 3)

	// Default range when omitted.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/links", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page))
	require.Len(t, page, 3)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/links?range=[5,1]", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRedirectAndListLinkVisits(t *testing.T) {
	store, router := newTestRouter()

	created, err := store.CreateLink(t.Context(), storage.CreateLinkInput{
		OriginalURL: "https://example.com/path",
		ShortURL:    "http://localhost:8080/r/demo",
		ShortName:   "demo",
	})
	require.NoError(t, err)

	// first redirect
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/r/demo", nil)
	req.Header.Set("User-Agent", "curl/8.5.0")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com/path", w.Header().Get("Location"))

	// second redirect
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/r/demo", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusFound, w.Code)

	// list all visits
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/link_visits?range=[0,10]", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "link_visits 0-1/2", w.Header().Get("Content-Range"))

	var visits []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &visits))
	require.Len(t, visits, 2)
	assert.EqualValues(t, created.ID, visits[0]["link_id"])
	assert.EqualValues(t, 302, visits[0]["status"])
	assert.NotEmpty(t, visits[0]["user_agent"])

	// filter by link_id
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/link_visits?link_id=1&range=[0,10]", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "link_visits 0-1/2", w.Header().Get("Content-Range"))

	// missing code
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/r/missing", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRedirectFailsOpenWhenVisitInsertFails(t *testing.T) {
	store, router := newTestRouter()

	_, err := store.CreateLink(t.Context(), storage.CreateLinkInput{
		OriginalURL: "https://example.com/fail-open",
		ShortURL:    "http://localhost:8080/r/fail",
		ShortName:   "fail",
	})
	require.NoError(t, err)

	store.SetCreateVisitError(errors.New("db write failed"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/r/fail", nil)
	router.ServeHTTP(w, req)

	// Redirect must still work even if analytics write fails.
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com/fail-open", w.Header().Get("Location"))
}

func TestListLinkVisitsInvalidLinkID(t *testing.T) {
	_, router := newTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/link_visits?link_id=abc&range=[0,10]", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/link_visits?link_id=-1&range=[0,10]", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/link_visits?link_id=0&range=[0,10]", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
