package api

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"code/internal/storage"

	"github.com/gin-gonic/gin"
)

func (s *Server) redirectByShortName(c *gin.Context) {
	shortName := strings.TrimSpace(c.Param("short_name"))
	if shortName == "" || strings.Contains(shortName, "/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid short name"})
		return
	}

	link, err := s.store.GetLinkByShortName(c.Request.Context(), shortName)
	if err != nil {
		writeStorageError(c, err)
		return
	}

	c.Redirect(http.StatusFound, link.OriginalURL)
}

func (s *Server) buildShortURL(shortName string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(s.cfg.ShortURL), "/")
	if base == "" {
		return "", errors.New("SHORT_URL is not configured")
	}
	return base + "/" + shortName, nil
}

func parseIDParam(c *gin.Context) (int32, bool) {
	raw := c.Param("id")
	id64, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || id64 <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return int32(id64), true
}

func validateLinkInput(originalURL, shortName string) (string, string, error) {
	originalURL = strings.TrimSpace(originalURL)
	shortName = strings.TrimSpace(shortName)

	if originalURL == "" {
		return "", "", errors.New("original_url is required")
	}
	if shortName == "" {
		return "", "", errors.New("short_name is required")
	}
	if strings.Contains(shortName, "/") {
		return "", "", errors.New("short_name must not contain '/'")
	}

	parsed, err := url.ParseRequestURI(originalURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", errors.New("original_url must be a valid absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", errors.New("original_url must use http or https")
	}

	return originalURL, shortName, nil
}

func writeStorageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, storage.ErrURLNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
	case errors.Is(err, storage.ErrURLAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "link already exists"})
	default:
		writeInternalError(c, err)
	}
}

func writeInternalError(c *gin.Context, err error) {
	log.Printf(
		"request failed method=%s path=%s status=500 err=%v",
		c.Request.Method,
		c.Request.URL.Path,
		err,
	)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func toLinkResponse(link storage.Link) linkResponse {
	return linkResponse{
		ID:          link.ID,
		OriginalURL: link.OriginalURL,
		ShortURL:    link.ShortURL,
		ShortName:   link.ShortName,
		CreatedAt:   link.CreatedAt,
	}
}
