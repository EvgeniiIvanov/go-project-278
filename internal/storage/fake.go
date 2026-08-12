package storage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Fake is an in-memory Storage implementation for tests.
type Fake struct {
	mu             sync.Mutex
	nextID         int32
	nextVisitID    int64
	links          map[int32]Link
	visits         []LinkVisit
	pingErr        error
	createVisitErr error
}

// NewFake returns an empty in-memory storage.
func NewFake() *Fake {
	return &Fake{
		nextID:      1,
		nextVisitID: 1,
		links:       make(map[int32]Link),
		visits:      make([]LinkVisit, 0),
	}
}

// SetCreateVisitError forces CreateLinkVisit to return err (for fail-closed tests).
func (f *Fake) SetCreateVisitError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createVisitErr = err
}

func (f *Fake) ListLinks(ctx context.Context, input ListLinksInput) (ListLinksResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if input.From < 0 || input.To < input.From {
		return ListLinksResult{}, fmt.Errorf("invalid list range: from=%d to=%d", input.From, input.To)
	}

	all := make([]Link, 0, len(f.links))
	for _, link := range f.links {
		all = append(all, link)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})

	total := int64(len(all))
	from := int(input.From)
	if from >= len(all) {
		return ListLinksResult{Links: []Link{}, Total: total}, nil
	}

	to := int(input.To) + 1
	if to > len(all) {
		to = len(all)
	}

	page := make([]Link, to-from)
	copy(page, all[from:to])
	return ListLinksResult{Links: page, Total: total}, nil
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

	filtered := f.visits[:0]
	for _, visit := range f.visits {
		if visit.LinkID != id {
			filtered = append(filtered, visit)
		}
	}
	f.visits = filtered
	return nil
}

func (f *Fake) CreateLinkVisit(ctx context.Context, input CreateLinkVisitInput) (LinkVisit, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.createVisitErr != nil {
		return LinkVisit{}, f.createVisitErr
	}
	if _, ok := f.links[input.LinkID]; !ok {
		return LinkVisit{}, ErrURLNotFound
	}

	visit := LinkVisit{
		ID:        f.nextVisitID,
		LinkID:    input.LinkID,
		CreatedAt: time.Now().UTC(),
		IP:        input.IP,
		UserAgent: input.UserAgent,
		Status:    input.Status,
	}
	f.nextVisitID++
	f.visits = append(f.visits, visit)
	return visit, nil
}

func (f *Fake) ListLinkVisits(ctx context.Context, input ListLinkVisitsInput) (ListLinkVisitsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if input.From < 0 || input.To < input.From {
		return ListLinkVisitsResult{}, fmt.Errorf("invalid list range: from=%d to=%d", input.From, input.To)
	}

	all := make([]LinkVisit, 0, len(f.visits))
	for _, visit := range f.visits {
		if input.LinkID != nil && visit.LinkID != *input.LinkID {
			continue
		}
		all = append(all, visit)
	}

	// Newest first, matching SQL ORDER BY id DESC.
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID > all[j].ID
	})

	total := int64(len(all))
	from := int(input.From)
	if from >= len(all) {
		return ListLinkVisitsResult{Visits: []LinkVisit{}, Total: total}, nil
	}

	to := int(input.To) + 1
	if to > len(all) {
		to = len(all)
	}

	page := make([]LinkVisit, to-from)
	copy(page, all[from:to])
	return ListLinkVisitsResult{Visits: page, Total: total}, nil
}

func (f *Fake) Ping(ctx context.Context) error {
	return f.pingErr
}

func (f *Fake) Close() {}

var _ Storage = (*Fake)(nil)
