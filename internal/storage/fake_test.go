package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeStorageCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewFake()

	created, err := s.CreateLink(ctx, CreateLinkInput{
		OriginalURL: "https://example.com",
		ShortURL:    "http://localhost:8080/abc",
		ShortName:   "abc",
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), created.ID)

	_, err = s.CreateLink(ctx, CreateLinkInput{ShortName: "abc"})
	assert.ErrorIs(t, err, ErrURLAlreadyExists)

	got, err := s.GetLinkByShortName(ctx, "abc")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	updated, err := s.UpdateLink(ctx, UpdateLinkInput{
		ID:          created.ID,
		OriginalURL: "https://example.org",
		ShortURL:    created.ShortURL,
		ShortName:   "abc",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.org", updated.OriginalURL)

	require.NoError(t, s.DeleteLink(ctx, created.ID))
	_, err = s.GetLinkByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrURLNotFound)
	assert.ErrorIs(t, s.DeleteLink(ctx, created.ID), ErrURLNotFound)
}

func TestFakeCreateLinkVisitUnknownLink(t *testing.T) {
	s := NewFake()

	_, err := s.CreateLinkVisit(context.Background(), CreateLinkVisitInput{
		LinkID:    999,
		IP:        "127.0.0.1",
		UserAgent: "test",
		Status:    302,
	})
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestFakeLinkVisitsFilterAndDeleteCascade(t *testing.T) {
	ctx := context.Background()
	s := NewFake()

	link1, err := s.CreateLink(ctx, CreateLinkInput{
		OriginalURL: "https://example.com/1",
		ShortURL:    "http://localhost:8080/r/one",
		ShortName:   "one",
	})
	require.NoError(t, err)
	link2, err := s.CreateLink(ctx, CreateLinkInput{
		OriginalURL: "https://example.com/2",
		ShortURL:    "http://localhost:8080/r/two",
		ShortName:   "two",
	})
	require.NoError(t, err)

	_, err = s.CreateLinkVisit(ctx, CreateLinkVisitInput{LinkID: link1.ID, IP: "1.1.1.1", Status: 302})
	require.NoError(t, err)
	_, err = s.CreateLinkVisit(ctx, CreateLinkVisitInput{LinkID: link2.ID, IP: "2.2.2.2", Status: 302})
	require.NoError(t, err)

	all, err := s.ListLinkVisits(ctx, ListLinkVisitsInput{From: 0, To: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), all.Total)

	linkID := link1.ID
	filtered, err := s.ListLinkVisits(ctx, ListLinkVisitsInput{LinkID: &linkID, From: 0, To: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), filtered.Total)
	require.Len(t, filtered.Visits, 1)
	assert.Equal(t, link1.ID, filtered.Visits[0].LinkID)

	require.NoError(t, s.DeleteLink(ctx, link1.ID))
	afterDelete, err := s.ListLinkVisits(ctx, ListLinkVisitsInput{From: 0, To: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), afterDelete.Total)
	assert.Equal(t, link2.ID, afterDelete.Visits[0].LinkID)
}
