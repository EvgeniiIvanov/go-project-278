package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code/internal/api"
	"code/internal/storage"

	"github.com/stretchr/testify/assert"
)

func TestPingRouteTableDriven(t *testing.T) {
	router := api.NewRouter(storage.NewFake(), testConfig())

	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
		wantBody string
	}{
		{"ping endpoint", "GET", "/ping", 200, "pong"},
		{"not found", "GET", "/nonexistent", 404, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, w.Body.String())
			}
		})
	}
}
