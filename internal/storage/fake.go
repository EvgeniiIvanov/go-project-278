package storage

import (
	"context"
	"sync"
	"time"
)

// Fake is an in-memory Storage implementation for tests.
type Fake struct {
	mu      sync.Mutex
	nextID  int32
	links   map[int32]Link
	pingErr error
}

// NewFake returns an empty in-memory storage.
func NewFake() *Fake {
	return &Fake{
		nextID: 1,
		links:  make(map[int32]Link),
	}
}

func (f *Fake) ListLinks(ctx context.Context) ([]Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := make([]Link, 0, len(f.links))
	for _, link := range f.links {
		result = append(result, link)
	}
	return result, nil
}

func (f *Fake) GetLinkByID(ctx context.Context, id int32) (Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	link, ok := f.links[id]
	if !ok {
		return Link{}, ErrURLNotFound
	}
	return link, nil
}

func (f *Fake) GetLinkByShortName(ctx context.Context, shortName string) (Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, link := range f.links {
		if link.ShortName == shortName {
			return link, nil
		}
	}
	return Link{}, ErrURLNotFound
}

func (f *Fake) CreateLink(ctx context.Context, input CreateLinkInput) (Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, link := range f.links {
		if link.ShortName == input.ShortName {
			return Link{}, ErrURLAlreadyExists
		}
	}

	link := Link{
		ID:          f.nextID,
		OriginalURL: input.OriginalURL,
		ShortURL:    input.ShortURL,
		ShortName:   input.ShortName,
		CreatedAt:   time.Now().UTC(),
	}
	f.links[link.ID] = link
	f.nextID++
	return link, nil
}

func (f *Fake) UpdateLink(ctx context.Context, input UpdateLinkInput) (Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	link, ok := f.links[input.ID]
	if !ok {
		return Link{}, ErrURLNotFound
	}

	for id, existing := range f.links {
		if id != input.ID && existing.ShortName == input.ShortName {
			return Link{}, ErrURLAlreadyExists
		}
	}

	link.OriginalURL = input.OriginalURL
	link.ShortURL = input.ShortURL
	link.ShortName = input.ShortName
	f.links[input.ID] = link
	return link, nil
}

func (f *Fake) DeleteLink(ctx context.Context, id int32) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.links[id]; !ok {
		return ErrURLNotFound
	}
	delete(f.links, id)
	return nil
}

func (f *Fake) Ping(ctx context.Context) error {
	return f.pingErr
}

func (f *Fake) Close() {}

var _ Storage = (*Fake)(nil)
