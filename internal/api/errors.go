package api

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"code/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	// Use JSON field names in validator messages:
	// createLinkPayload.original_url instead of createLinkPayload.OriginalURL
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.Split(fld.Tag.Get("json"), ",")[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})
	}
}

// Field validation errors:
// 422 { "errors": { "<field>": "<message>" } }
type fieldErrorsResponse struct {
	Errors map[string]string `json:"errors"`
}

// Simple message errors:
// 400/404/500 { "error": "<message>" }
type plainErrorResponse struct {
	Error string `json:"error"`
}

func writeError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, plainErrorResponse{Error: message})
}

func writeFieldErrors(c *gin.Context, fields map[string]string) {
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, fieldErrorsResponse{Errors: fields})
}

func writeInvalidRequest(c *gin.Context) {
	writeError(c, http.StatusBadRequest, "invalid request")
}

func writeValidationError(c *gin.Context, message string) {
	writeError(c, http.StatusBadRequest, message)
}

func writeNotFound(c *gin.Context, message string) {
	writeError(c, http.StatusNotFound, message)
}

func writeInternalError(c *gin.Context, err error) {
	slog.Error(
		"request failed",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"status", http.StatusInternalServerError,
		"err", err,
	)
	writeError(c, http.StatusInternalServerError, "internal server error")
}

func writeStorageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, storage.ErrURLNotFound):
		writeNotFound(c, "link not found")
	case errors.Is(err, storage.ErrURLAlreadyExists):
		writeFieldErrors(c, map[string]string{
			"short_name": "short name already in use",
		})
	default:
		writeInternalError(c, err)
	}
}

// bindJSON binds JSON body and writes the proper error response on failure.
// Returns false when the handler should stop.
func bindJSON(c *gin.Context, dst any) bool {
	err := c.ShouldBindBodyWith(dst, binding.JSON)
	if err == nil {
		return true
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make(map[string]string, len(ve))
		for _, fe := range ve {
			fields[fieldName(fe)] = fe.Error()
		}
		writeFieldErrors(c, fields)
		return false
	}

	// Malformed JSON, empty body, type mismatches, etc.
	writeInvalidRequest(c)
	return false
}

func fieldName(fe validator.FieldError) string {
	// With RegisterTagNameFunc, Field() is already the JSON name.
	name := fe.Field()
	switch name {
	case "OriginalURL":
		return "original_url"
	case "ShortName":
		return "short_name"
	default:
		return name
	}
}
