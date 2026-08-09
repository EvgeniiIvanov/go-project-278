package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	linksdb "code/internal/db/links"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns          = int32(10)
	defaultMinConns          = int32(1)
	defaultMaxConnLifetime   = 30 * time.Minute
	defaultMaxConnIdleTime   = 5 * time.Minute
	defaultHealthCheckPeriod = 1 * time.Minute
)

// Postgres is a Postgres-backed Storage implementation.
type Postgres struct {
	pool *pgxpool.Pool
	q    linksdb.Querier
}

// NewPostgres opens a connection pool, pings the database, and returns Storage.
func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is empty")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	cfg.MaxConns = int32FromEnv("DB_MAX_CONNS", defaultMaxConns)
	cfg.MinConns = int32FromEnv("DB_MIN_CONNS", defaultMinConns)
	cfg.MaxConnLifetime = durationFromEnv("DB_MAX_CONN_LIFETIME", defaultMaxConnLifetime)
	cfg.MaxConnIdleTime = durationFromEnv("DB_MAX_CONN_IDLE_TIME", defaultMaxConnIdleTime)
	cfg.HealthCheckPeriod = durationFromEnv("DB_HEALTHCHECK_PERIOD", defaultHealthCheckPeriod)

	if cfg.MinConns > cfg.MaxConns {
		return nil, fmt.Errorf("DB_MIN_CONNS (%d) cannot be greater than DB_MAX_CONNS (%d)", cfg.MinConns, cfg.MaxConns)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open db pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &Postgres{
		pool: pool,
		q:    linksdb.New(pool),
	}, nil
}

// newPostgresWithQuerier is used by unit tests to inject a fake querier.
func newPostgresWithQuerier(q linksdb.Querier) *Postgres {
	return &Postgres{q: q}
}

func int32FromEnv(key string, fallback int32) int32 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	v, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || v < 0 {
		return fallback
	}
	return int32(v)
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d < 0 {
		return fallback
	}
	return d
}

func (p *Postgres) ListLinks(ctx context.Context) ([]Link, error) {
	rows, err := p.q.ListLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}

	links := make([]Link, 0, len(rows))
	for _, row := range rows {
		links = append(links, toLink(row))
	}
	return links, nil
}

func (p *Postgres) GetLinkByID(ctx context.Context, id int32) (Link, error) {
	row, err := p.q.GetLinkByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrURLNotFound
		}
		return Link{}, fmt.Errorf("get link by id: %w", err)
	}
	return toLink(row), nil
}

func (p *Postgres) GetLinkByShortName(ctx context.Context, shortName string) (Link, error) {
	row, err := p.q.GetLinkByShortName(ctx, shortName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrURLNotFound
		}
		return Link{}, fmt.Errorf("get link by short name: %w", err)
	}
	return toLink(row), nil
}

func (p *Postgres) CreateLink(ctx context.Context, input CreateLinkInput) (Link, error) {
	row, err := p.q.CreateLink(ctx, linksdb.CreateLinkParams{
		OriginalUrl: input.OriginalURL,
		ShortUrl:    input.ShortURL,
		ShortName:   input.ShortName,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Link{}, ErrURLAlreadyExists
		}
		return Link{}, fmt.Errorf("create link: %w", err)
	}
	return toLink(row), nil
}

func (p *Postgres) UpdateLink(ctx context.Context, input UpdateLinkInput) (Link, error) {
	row, err := p.q.UpdateLink(ctx, linksdb.UpdateLinkParams{
		OriginalUrl: input.OriginalURL,
		ShortUrl:    input.ShortURL,
		ShortName:   input.ShortName,
		ID:          input.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrURLNotFound
		}
		if isUniqueViolation(err) {
			return Link{}, ErrURLAlreadyExists
		}
		return Link{}, fmt.Errorf("update link: %w", err)
	}
	return toLink(row), nil
}

func (p *Postgres) DeleteLink(ctx context.Context, id int32) error {
	n, err := p.q.DeleteLink(ctx, id)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	if n == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *Postgres) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

func toLink(row linksdb.Link) Link {
	var createdAt time.Time
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}

	return Link{
		ID:          row.ID,
		OriginalURL: row.OriginalUrl,
		ShortURL:    row.ShortUrl,
		ShortName:   row.ShortName,
		CreatedAt:   createdAt,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Ensure Postgres satisfies Storage at compile time.
var _ Storage = (*Postgres)(nil)
