package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"code/internal/storage"

	"github.com/gin-gonic/gin"
)

type Server struct {
	store storage.Storage
	cfg   Config
}

type createLinkRequest struct {
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
}

type updateLinkRequest struct {
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
}

type linkResponse struct {
	ID          int32     `json:"id"`
	OriginalURL string    `json:"original_url"`
	ShortURL    string    `json:"short_url"`
	ShortName   string    `json:"short_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type linkVisitResponse struct {
	ID        int64     `json:"id"`
	LinkID    int32     `json:"link_id"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Status    int32     `json:"status"`
}

func (s *Server) ping(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func (s *Server) listLinks(c *gin.Context) {
	from, to, err := parseRangeQuery(c.Query("range"))
	if err != nil {
		writeValidationError(c, err.Error())
		return
	}

	result, err := s.store.ListLinks(c.Request.Context(), storage.ListLinksInput{
		From: from,
		To:   to,
	})
	if err != nil {
		writeStorageError(c, err)
		return
	}

	resp := make([]linkResponse, 0, len(result.Links))
	for _, link := range result.Links {
		resp = append(resp, toLinkResponse(link))
	}

	// Content-Range uses inclusive indexes: links <from>-<to>/<total>
	// If the page is empty, still report the requested from and total.
	end := from
	if len(result.Links) > 0 {
		end = from + int32(len(result.Links)) - 1
	}
	c.Header("Content-Range", formatContentRange("links", from, end, result.Total))
	c.JSON(http.StatusOK, resp)
}

func (s *Server) listLinkVisits(c *gin.Context) {
	from, to, err := parseRangeQuery(c.Query("range"))
	if err != nil {
		writeValidationError(c, err.Error())
		return
	}

	input := storage.ListLinkVisitsInput{
		From: from,
		To:   to,
	}

	if rawLinkID := strings.TrimSpace(c.Query("link_id")); rawLinkID != "" {
		linkID, err := strconv.ParseInt(rawLinkID, 10, 32)
		if err != nil || linkID <= 0 {
			writeValidationError(c, "link_id must be a positive integer")
			return
		}
		id := int32(linkID)
		input.LinkID = &id
	}

	result, err := s.store.ListLinkVisits(c.Request.Context(), input)
	if err != nil {
		writeStorageError(c, err)
		return
	}

	resp := make([]linkVisitResponse, 0, len(result.Visits))
	for _, visit := range result.Visits {
		resp = append(resp, toLinkVisitResponse(visit))
	}

	end := from
	if len(result.Visits) > 0 {
		end = from + int32(len(result.Visits)) - 1
	}
	c.Header("Content-Range", formatContentRange("link_visits", from, end, result.Total))
	c.JSON(http.StatusOK, resp)
}

func (s *Server) getLinkByID(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	link, err := s.store.GetLinkByID(c.Request.Context(), id)
	if err != nil {
		writeStorageError(c, err)
		return
	}
	c.JSON(http.StatusOK, toLinkResponse(link))
}

func (s *Server) createLink(c *gin.Context) {
	var req createLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeInvalidJSON(c)
		return
	}

	originalURL, shortName, err := validateLinkInput(req.OriginalURL, req.ShortName)
	if err != nil {
		writeValidationError(c, err.Error())
		return
	}

	shortURL, err := s.buildShortURL(shortName)
	if err != nil {
		writeInternalError(c, err)
		return
	}

	link, err := s.store.CreateLink(c.Request.Context(), storage.CreateLinkInput{
		OriginalURL: originalURL,
		ShortURL:    shortURL,
		ShortName:   shortName,
	})
	if err != nil {
		writeStorageError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toLinkResponse(link))
}

func (s *Server) updateLink(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	var req updateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeInvalidJSON(c)
		return
	}

	originalURL, shortName, err := validateLinkInput(req.OriginalURL, req.ShortName)
	if err != nil {
		writeValidationError(c, err.Error())
		return
	}

	shortURL, err := s.buildShortURL(shortName)
	if err != nil {
		writeInternalError(c, err)
		return
	}

	link, err := s.store.UpdateLink(c.Request.Context(), storage.UpdateLinkInput{
		ID:          id,
		OriginalURL: originalURL,
		ShortURL:    shortURL,
		ShortName:   shortName,
	})
	if err != nil {
		writeStorageError(c, err)
		return
	}
	c.JSON(http.StatusOK, toLinkResponse(link))
}

func (s *Server) deleteLink(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}

	if err := s.store.DeleteLink(c.Request.Context(), id); err != nil {
		writeStorageError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
