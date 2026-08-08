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

	err = s.UpdateLink(ctx, UpdateLinkInput{
		ID:          created.ID,
		OriginalURL: "https://example.org",
		ShortURL:    created.ShortURL,
		ShortName:   "abc",
	})
	require.NoError(t, err)

	gotByID, err := s.GetLinkByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.org", gotByID.OriginalURL)

	require.NoError(t, s.DeleteLink(ctx, created.ID))
	_, err = s.GetLinkByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrURLNotFound)
	assert.ErrorIs(t, s.DeleteLink(ctx, created.ID), ErrURLNotFound)
}
