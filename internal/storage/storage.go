package storage

import (
	"context"
	"errors"
	"time"
)

var (
	ErrURLNotFound      = errors.New("url not found")
	ErrURLAlreadyExists = errors.New("url already exists")
)

// Link is the app-facing representation of a shortened URL.
type Link struct {
	ID          int32
	OriginalURL string
	ShortURL    string
	ShortName   string
	CreatedAt   time.Time
}

// CreateLinkInput holds data required to store a new link.
type CreateLinkInput struct {
	OriginalURL string
	ShortURL    string
	ShortName   string
}

// UpdateLinkInput holds data required to update an existing link.
type UpdateLinkInput struct {
	ID          int32
	OriginalURL string
	ShortURL    string
	ShortName   string
}

// ListLinksInput holds inclusive range pagination for listing links.
// From/To are zero-based inclusive indexes, matching ?range=[from,to].
type ListLinksInput struct {
	From int32
	To   int32
}

// ListLinksResult is a page of links plus the total number of rows.
type ListLinksResult struct {
	Links []Link
	Total int64
}

// Storage is the app-facing persistence API.
type Storage interface {
	ListLinks(ctx context.Context, input ListLinksInput) (ListLinksResult, error)
	GetLinkByID(ctx context.Context, id int32) (Link, error)
	GetLinkByShortName(ctx context.Context, shortName string) (Link, error)
	CreateLink(ctx context.Context, input CreateLinkInput) (Link, error)
	UpdateLink(ctx context.Context, input UpdateLinkInput) (Link, error)
	DeleteLink(ctx context.Context, id int32) error
	Ping(ctx context.Context) error
	Close()
}
