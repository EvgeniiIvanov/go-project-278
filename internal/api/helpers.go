package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"code/internal/storage"

	"github.com/gin-gonic/gin"
)

const (
	defaultRangeFrom int32 = 0
	defaultRangeTo   int32 = 9
	maxRangeWindow   int32 = 100

	// 62^8 ~= 2.18e14 combinations; collisions are rare, but still possible.
	generatedShortNameLen     = 8
	generatedShortNameRetries = 5
	shortNameAlphabet         = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// parseRangeQuery parses an inclusive range query like "[0,10]" or "0,10".
// Empty value defaults to [0,9].
func parseRangeQuery(raw string) (int32, int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultRangeFrom, defaultRangeTo, nil
	}

	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	raw = strings.TrimSpace(raw)

	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return 0, 0, errors.New(`range must look like "[0,10]"`)
	}

	from64, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 32)
	if err != nil {
		return 0, 0, errors.New("range from must be an integer")
	}
	to64, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
	if err != nil {
		return 0, 0, errors.New("range to must be an integer")
	}

	from := int32(from64)
	to := int32(to64)
	if from < 0 {
		return 0, 0, errors.New("range from must be >= 0")
	}
	if to < from {
		return 0, 0, errors.New("range to must be >= from")
	}
	if to-from+1 > maxRangeWindow {
		return 0, 0, fmt.Errorf("range window must be <= %d", maxRangeWindow)
	}

	return from, to, nil
}

func formatContentRange(unit string, from, to int32, total int64) string {
	return fmt.Sprintf("%s %d-%d/%d", unit, from, to, total)
}

func (s *Server) redirectByCode(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" || strings.Contains(code, "/") {
		writeValidationError(c, "invalid code")
		return
	}

	link, err := s.store.GetLinkByShortName(c.Request.Context(), code)
	if err != nil {
		writeStorageError(c, err)
		return
	}

	status := http.StatusFound
	_, err = s.store.CreateLinkVisit(c.Request.Context(), storage.CreateLinkVisitInput{
		LinkID:    link.ID,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    int32(status),
	})
	if err != nil {
		// Fail closed: do not redirect if visit could not be recorded.
		writeStorageError(c, err)
		return
	}

	c.Redirect(status, link.OriginalURL)
}

func (s *Server) buildShortURL(shortName string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(s.cfg.ShortURL), "/")
	if base == "" {
		return "", errors.New("SHORT_URL is not configured")
	}
	return base + "/r/" + shortName, nil
}

func parseIDParam(c *gin.Context) (int32, bool) {
	raw := c.Param("id")
	id64, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || id64 <= 0 {
		writeValidationError(c, "invalid id")
		return 0, false
	}
	return int32(id64), true
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

func toLinkVisitResponse(visit storage.LinkVisit) linkVisitResponse {
	return linkVisitResponse{
		ID:        visit.ID,
		LinkID:    visit.LinkID,
		CreatedAt: visit.CreatedAt,
		IP:        visit.IP,
		UserAgent: visit.UserAgent,
		Status:    visit.Status,
	}
}

// generateShortNameFn is the short-name generator used by createLink.
// Tests may replace it to force deterministic collision/retry behavior.
var generateShortNameFn = defaultGenerateShortName

// defaultGenerateShortName returns a cryptographically random base62 string.
func defaultGenerateShortName(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("short name length must be positive")
	}

	var b strings.Builder
	b.Grow(length)

	max := big.NewInt(int64(len(shortNameAlphabet)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate short name: %w", err)
		}
		b.WriteByte(shortNameAlphabet[n.Int64()])
	}
	return b.String(), nil
}
