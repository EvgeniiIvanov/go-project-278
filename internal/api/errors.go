package api

import (
	"errors"
	"log/slog"
	"net/http"

	"code/internal/storage"

	"github.com/gin-gonic/gin"
)

// Stable error codes for clients. Messages may change; codes should not.
const (
	CodeInvalidJSON   = "invalid_json"
	CodeValidation    = "validation_error"
	CodeNotFound      = "not_found"
	CodeConflict      = "conflict"
	CodeInternalError = "internal_error"
)

// errorBody is the only error shape returned to clients.
// It must never include internal/implementation details.
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}

func writeValidationError(c *gin.Context, message string) {
	writeError(c, http.StatusBadRequest, CodeValidation, message)
}

func writeInvalidJSON(c *gin.Context) {
	writeError(c, http.StatusBadRequest, CodeInvalidJSON, "invalid JSON body")
}

func writeNotFound(c *gin.Context, message string) {
	writeError(c, http.StatusNotFound, CodeNotFound, message)
}

func writeConflict(c *gin.Context, message string) {
	writeError(c, http.StatusConflict, CodeConflict, message)
}

// writeInternalError logs the real error server-side and returns a generic payload.
func writeInternalError(c *gin.Context, err error) {
	slog.Error(
		"request failed",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", http.StatusInternalServerError,
		"err", err,
	)
	writeError(c, http.StatusInternalServerError, CodeInternalError, "internal server error")
}

// writeStorageError maps known domain errors to safe client errors.
// Unknown errors are treated as internal and never exposed.
func writeStorageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, storage.ErrURLNotFound):
		writeNotFound(c, "link not found")
	case errors.Is(err, storage.ErrURLAlreadyExists):
		writeConflict(c, "link already exists")
	default:
		writeInternalError(c, err)
	}
}
