package api

import (
	"errors"
	"io"
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
	// Make validation namespaces use JSON field names:
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

// Stable error codes for non-field API errors.
const (
	CodeNotFound      = "not_found"
	CodeInternalError = "internal_error"
)

// Hexlet field-validation contract:
// 422 { "errors": { "<field>": "<message>" } }
type fieldErrorsResponse struct {
	Errors map[string]string `json:"errors"`
}

// Hexlet invalid JSON contract:
// 400 { "error": "invalid request" }
type plainErrorResponse struct {
	Error string `json:"error"`
}

type codeMessageError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type codeMessageErrorResponse struct {
	Error codeMessageError `json:"error"`
}

func writeFieldErrors(c *gin.Context, fields map[string]string) {
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, fieldErrorsResponse{Errors: fields})
}

func writeInvalidRequest(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, plainErrorResponse{Error: "invalid request"})
}

func writeValidationError(c *gin.Context, message string) {
	// Non-field/query validation still uses a simple 400 payload.
	c.AbortWithStatusJSON(http.StatusBadRequest, plainErrorResponse{Error: message})
}

func writeNotFound(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusNotFound, codeMessageErrorResponse{
		Error: codeMessageError{Code: CodeNotFound, Message: message},
	})
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
	c.AbortWithStatusJSON(http.StatusInternalServerError, codeMessageErrorResponse{
		Error: codeMessageError{Code: CodeInternalError, Message: "internal server error"},
	})
}

// writeStorageError maps known domain errors to client responses.
// Unique conflicts on links are field errors (short_name), as required by Hexlet.
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

// bindJSONPayload binds JSON and maps binding errors to Hexlet response shapes.
// Returns false when response was already written.
func bindJSONPayload(c *gin.Context, dst any) bool {
	if err := c.ShouldBindBodyWith(dst, binding.JSON); err != nil {
		writeBindError(c, err)
		return false
	}
	return true
}

func writeBindError(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	switch {
	case errors.As(err, &ve):
		fields := make(map[string]string, len(ve))
		for _, fe := range ve {
			field := jsonFieldName(fe)
			fields[field] = fe.Error()
		}
		writeFieldErrors(c, fields)
	case errors.Is(err, io.EOF):
		writeInvalidRequest(c)
	default:
		// Malformed JSON / type mismatch / empty body syntax errors.
		writeInvalidRequest(c)
	}
}

func jsonFieldName(fe validator.FieldError) string {
	// Prefer JSON name if available; fallback to lower-snake-ish namespace.
	name := fe.Field()
	if ns := fe.Namespace(); ns != "" {
		// Namespace looks like createLinkPayload.OriginalURL or createLinkPayload.original_url
		if i := strings.LastIndex(ns, "."); i >= 0 && i+1 < len(ns) {
			name = ns[i+1:]
		}
	}
	switch name {
	case "OriginalURL", "original_url":
		return "original_url"
	case "ShortName", "short_name":
		return "short_name"
	default:
		return toSnakeCase(name)
	}
}

func toSnakeCase(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
