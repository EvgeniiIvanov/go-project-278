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
	const op = "storage.postgres.NewPostgres"

	if databaseURL == "" {
		return nil, fmt.Errorf("%s: database URL is empty", op)
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("%s: parse db config: %w", op, err)
	}

	cfg.MaxConns = int32FromEnv("DB_MAX_CONNS", defaultMaxConns)
	cfg.MinConns = int32FromEnv("DB_MIN_CONNS", defaultMinConns)
	cfg.MaxConnLifetime = durationFromEnv("DB_MAX_CONN_LIFETIME", defaultMaxConnLifetime)
	cfg.MaxConnIdleTime = durationFromEnv("DB_MAX_CONN_IDLE_TIME", defaultMaxConnIdleTime)
	cfg.HealthCheckPeriod = durationFromEnv("DB_HEALTHCHECK_PERIOD", defaultHealthCheckPeriod)

	if cfg.MinConns > cfg.MaxConns {
		return nil, fmt.Errorf("%s: DB_MIN_CONNS (%d) cannot be greater than DB_MAX_CONNS (%d)", op, cfg.MinConns, cfg.MaxConns)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: open db pool: %w", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%s: ping db: %w", op, err)
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

func (p *Postgres) ListLinks(ctx context.Context, input ListLinksInput) (ListLinksResult, error) {
	const op = "storage.postgres.ListLinks"
	if input.From < 0 || input.To < input.From {
		return ListLinksResult{}, fmt.Errorf("%s: invalid list range: from=%d to=%d", op, input.From, input.To)
	}

	limit := input.To - input.From + 1
	rows, err := p.q.ListLinks(ctx, linksdb.ListLinksParams{
		Limit:  limit,
		Offset: input.From,
	})
	if err != nil {
		return ListLinksResult{}, fmt.Errorf("%s: %w", op, err)
	}

	total, err := p.q.CountLinks(ctx)
	if err != nil {
		return ListLinksResult{}, fmt.Errorf("%s: count: %w", op, err)
	}

	links := make([]Link, 0, len(rows))
	for _, row := range rows {
		links = append(links, toLink(row))
	}
	return ListLinksResult{Links: links, Total: total}, nil
}

func (p *Postgres) GetLinkByID(ctx context.Context, id int32) (Link, error) {
	const op = "storage.postgres.GetLinkByID"
	row, err := p.q.GetLinkByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrURLNotFound
		}
		return Link{}, fmt.Errorf("%s: %w", op, err)
	}
	return toLink(row), nil
}

func (p *Postgres) GetLinkByShortName(ctx context.Context, shortName string) (Link, error) {
	const op = "storage.postgres.GetLinkByShortName"
	row, err := p.q.GetLinkByShortName(ctx, shortName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Link{}, ErrURLNotFound
		}
		return Link{}, fmt.Errorf("%s: %w", op, err)
	}
	return toLink(row), nil
}

func (p *Postgres) CreateLink(ctx context.Context, input CreateLinkInput) (Link, error) {
	const op = "storage.postgres.CreateLink"
	row, err := p.q.CreateLink(ctx, linksdb.CreateLinkParams{
		OriginalUrl: input.OriginalURL,
		ShortUrl:    input.ShortURL,
		ShortName:   input.ShortName,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Link{}, ErrURLAlreadyExists
		}
		return Link{}, fmt.Errorf("%s: %w", op, err)
	}
	return toLink(row), nil
}

func (p *Postgres) UpdateLink(ctx context.Context, input UpdateLinkInput) (Link, error) {
	const op = "storage.postgres.UpdateLink"
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
		return Link{}, fmt.Errorf("%s: %w", op, err)
	}
	return toLink(row), nil
}

func (p *Postgres) DeleteLink(ctx context.Context, id int32) error {
	const op = "storage.postgres.DeleteLink"
	n, err := p.q.DeleteLink(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if n == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (p *Postgres) CreateLinkVisit(ctx context.Context, input CreateLinkVisitInput) (LinkVisit, error) {
	const op = "storage.postgres.CreateLinkVisit"
	row, err := p.q.CreateLinkVisit(ctx, linksdb.CreateLinkVisitParams{
		LinkID:    input.LinkID,
		Ip:        input.IP,
		UserAgent: input.UserAgent,
		Status:    input.Status,
	})
	if err != nil {
		return LinkVisit{}, fmt.Errorf("%s: %w", op, err)
	}
	return toLinkVisit(row), nil
}

func (p *Postgres) ListLinkVisits(ctx context.Context, input ListLinkVisitsInput) (ListLinkVisitsResult, error) {
	const op = "storage.postgres.ListLinkVisits"
	if input.From < 0 || input.To < input.From {
		return ListLinkVisitsResult{}, fmt.Errorf("%s: invalid list range: from=%d to=%d", op, input.From, input.To)
	}

	limit := input.To - input.From + 1
	offset := input.From

	var (
		rows  []linksdb.LinkVisit
		total int64
		err   error
	)

	if input.LinkID != nil {
		rows, err = p.q.ListLinkVisitsByLinkID(ctx, linksdb.ListLinkVisitsByLinkIDParams{
			LinkID: *input.LinkID,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return ListLinkVisitsResult{}, fmt.Errorf("%s: %w", op, err)
		}
		total, err = p.q.CountLinkVisitsByLinkID(ctx, *input.LinkID)
		if err != nil {
			return ListLinkVisitsResult{}, fmt.Errorf("%s: count by link id: %w", op, err)
		}
	} else {
		rows, err = p.q.ListLinkVisits(ctx, linksdb.ListLinkVisitsParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return ListLinkVisitsResult{}, fmt.Errorf("%s: %w", op, err)
		}
		total, err = p.q.CountLinkVisits(ctx)
		if err != nil {
			return ListLinkVisitsResult{}, fmt.Errorf("%s: count: %w", op, err)
		}
	}

	visits := make([]LinkVisit, 0, len(rows))
	for _, row := range rows {
		visits = append(visits, toLinkVisit(row))
	}
	return ListLinkVisitsResult{Visits: visits, Total: total}, nil
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

func toLinkVisit(row linksdb.LinkVisit) LinkVisit {
	var createdAt time.Time
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}

	return LinkVisit{
		ID:        row.ID,
		LinkID:    row.LinkID,
		CreatedAt: createdAt,
		IP:        row.Ip,
		UserAgent: row.UserAgent,
		Status:    row.Status,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Ensure Postgres satisfies Storage at compile time.
var _ Storage = (*Postgres)(nil)
