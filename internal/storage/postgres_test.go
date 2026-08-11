package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	linksdb "code/internal/db/links"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolConfigEnvHelpers(t *testing.T) {
	t.Setenv("DB_MAX_CONNS", "20")
	assert.Equal(t, int32(20), int32FromEnv("DB_MAX_CONNS", defaultMaxConns))

	t.Setenv("DB_MAX_CONNS", "bad")
	assert.Equal(t, defaultMaxConns, int32FromEnv("DB_MAX_CONNS", defaultMaxConns))

	t.Setenv("DB_MAX_CONN_LIFETIME", "45m")
	assert.Equal(t, 45*time.Minute, durationFromEnv("DB_MAX_CONN_LIFETIME", defaultMaxConnLifetime))

	t.Setenv("DB_MAX_CONN_LIFETIME", "nope")
	assert.Equal(t, defaultMaxConnLifetime, durationFromEnv("DB_MAX_CONN_LIFETIME", defaultMaxConnLifetime))
}

type fakeQuerier struct {
	listLinksFn          func(ctx context.Context, arg linksdb.ListLinksParams) ([]linksdb.Link, error)
	countLinksFn         func(ctx context.Context) (int64, error)
	getLinkByIDFn        func(ctx context.Context, id int32) (linksdb.Link, error)
	getLinkByShortNameFn func(ctx context.Context, shortName string) (linksdb.Link, error)
	createLinkFn         func(ctx context.Context, arg linksdb.CreateLinkParams) (linksdb.Link, error)
	updateLinkFn         func(ctx context.Context, arg linksdb.UpdateLinkParams) (linksdb.Link, error)
	deleteLinkFn         func(ctx context.Context, id int32) (int64, error)
}

func (f *fakeQuerier) ListLinks(ctx context.Context, arg linksdb.ListLinksParams) ([]linksdb.Link, error) {
	return f.listLinksFn(ctx, arg)
}
func (f *fakeQuerier) CountLinks(ctx context.Context) (int64, error) {
	return f.countLinksFn(ctx)
}
func (f *fakeQuerier) GetLinkByID(ctx context.Context, id int32) (linksdb.Link, error) {
	return f.getLinkByIDFn(ctx, id)
}
func (f *fakeQuerier) GetLinkByShortName(ctx context.Context, shortName string) (linksdb.Link, error) {
	return f.getLinkByShortNameFn(ctx, shortName)
}
func (f *fakeQuerier) CreateLink(ctx context.Context, arg linksdb.CreateLinkParams) (linksdb.Link, error) {
	return f.createLinkFn(ctx, arg)
}
func (f *fakeQuerier) UpdateLink(ctx context.Context, arg linksdb.UpdateLinkParams) (linksdb.Link, error) {
	return f.updateLinkFn(ctx, arg)
}
func (f *fakeQuerier) DeleteLink(ctx context.Context, id int32) (int64, error) {
	return f.deleteLinkFn(ctx, id)
}

func sampleDBLink() linksdb.Link {
	return linksdb.Link{
		ID:          1,
		OriginalUrl: "https://example.com",
		ShortUrl:    "http://localhost:8080/abc",
		ShortName:   "abc",
		CreatedAt:   pgtype.Timestamp{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Valid: true},
	}
}

func TestGetLinkByID_NotFound(t *testing.T) {
	p := newPostgresWithQuerier(&fakeQuerier{
		getLinkByIDFn: func(ctx context.Context, id int32) (linksdb.Link, error) {
			return linksdb.Link{}, pgx.ErrNoRows
		},
	})

	_, err := p.GetLinkByID(context.Background(), 99)
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestGetLinkByShortName_NotFound(t *testing.T) {
	p := newPostgresWithQuerier(&fakeQuerier{
		getLinkByShortNameFn: func(ctx context.Context, shortName string) (linksdb.Link, error) {
			return linksdb.Link{}, pgx.ErrNoRows
		},
	})

	_, err := p.GetLinkByShortName(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestCreateLink_UniqueViolation(t *testing.T) {
	p := newPostgresWithQuerier(&fakeQuerier{
		createLinkFn: func(ctx context.Context, arg linksdb.CreateLinkParams) (linksdb.Link, error) {
			return linksdb.Link{}, &pgconn.PgError{Code: "23505"}
		},
	})

	_, err := p.CreateLink(context.Background(), CreateLinkInput{
		OriginalURL: "https://example.com",
		ShortURL:    "http://localhost:8080/abc",
		ShortName:   "abc",
	})
	assert.ErrorIs(t, err, ErrURLAlreadyExists)
}

func TestUpdateLink_NotFoundAndUnique(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		p := newPostgresWithQuerier(&fakeQuerier{
			updateLinkFn: func(ctx context.Context, arg linksdb.UpdateLinkParams) (linksdb.Link, error) {
				return linksdb.Link{}, pgx.ErrNoRows
			},
		})
		_, err := p.UpdateLink(context.Background(), UpdateLinkInput{ID: 1, ShortName: "a"})
		assert.ErrorIs(t, err, ErrURLNotFound)
	})

	t.Run("unique violation", func(t *testing.T) {
		p := newPostgresWithQuerier(&fakeQuerier{
			updateLinkFn: func(ctx context.Context, arg linksdb.UpdateLinkParams) (linksdb.Link, error) {
				return linksdb.Link{}, &pgconn.PgError{Code: "23505"}
			},
		})
		_, err := p.UpdateLink(context.Background(), UpdateLinkInput{ID: 1, ShortName: "a"})
		assert.ErrorIs(t, err, ErrURLAlreadyExists)
	})
}

func TestDeleteLink_NotFound(t *testing.T) {
	p := newPostgresWithQuerier(&fakeQuerier{
		deleteLinkFn: func(ctx context.Context, id int32) (int64, error) {
			return 0, nil
		},
	})

	err := p.DeleteLink(context.Background(), 99)
	assert.ErrorIs(t, err, ErrURLNotFound)
}

func TestListAndGet_SuccessMapping(t *testing.T) {
	dbLink := sampleDBLink()
	p := newPostgresWithQuerier(&fakeQuerier{
		listLinksFn: func(ctx context.Context, arg linksdb.ListLinksParams) ([]linksdb.Link, error) {
			assert.Equal(t, int32(10), arg.Limit)
			assert.Equal(t, int32(0), arg.Offset)
			return []linksdb.Link{dbLink}, nil
		},
		countLinksFn: func(ctx context.Context) (int64, error) {
			return 1, nil
		},
		getLinkByIDFn: func(ctx context.Context, id int32) (linksdb.Link, error) {
			return dbLink, nil
		},
	})

	result, err := p.ListLinks(context.Background(), ListLinksInput{From: 0, To: 9})
	require.NoError(t, err)
	require.Len(t, result.Links, 1)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, int32(1), result.Links[0].ID)
	assert.Equal(t, "abc", result.Links[0].ShortName)
	assert.Equal(t, dbLink.CreatedAt.Time, result.Links[0].CreatedAt)

	got, err := p.GetLinkByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got.OriginalURL)
}

func TestCreateLink_PassesThroughUnexpectedError(t *testing.T) {
	p := newPostgresWithQuerier(&fakeQuerier{
		createLinkFn: func(ctx context.Context, arg linksdb.CreateLinkParams) (linksdb.Link, error) {
			return linksdb.Link{}, errors.New("db down")
		},
	})

	_, err := p.CreateLink(context.Background(), CreateLinkInput{ShortName: "x"})
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrURLAlreadyExists)
	assert.NotErrorIs(t, err, ErrURLNotFound)
}
