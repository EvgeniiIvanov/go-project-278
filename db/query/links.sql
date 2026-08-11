-- name: ListLinks :many
SELECT * FROM links
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: CountLinks :one
SELECT count(*) FROM links;

-- name: GetLinkByID :one
SELECT * FROM links WHERE id = $1;

-- name: GetLinkByShortName :one
SELECT * FROM links WHERE short_name = $1;

-- name: CreateLink :one
INSERT INTO links (original_url, short_url, short_name) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateLink :one
UPDATE links SET original_url = $1, short_url = $2, short_name = $3 WHERE id = $4 RETURNING *;

-- name: DeleteLink :execrows
DELETE FROM links WHERE id = $1;