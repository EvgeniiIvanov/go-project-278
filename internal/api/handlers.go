package api

import (
	"net/http"
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

func (s *Server) ping(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

func (s *Server) listLinks(c *gin.Context) {
	links, err := s.store.ListLinks(c.Request.Context())
	if err != nil {
		writeStorageError(c, err)
		return
	}

	resp := make([]linkResponse, 0, len(links))
	for _, link := range links {
		resp = append(resp, toLinkResponse(link))
	}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	originalURL, shortName, err := validateLinkInput(req.OriginalURL, req.ShortName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	originalURL, shortName, err := validateLinkInput(req.OriginalURL, req.ShortName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
