-- name: CreateLinkVisit :one
INSERT INTO link_visits (link_id, ip, user_agent, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListLinkVisits :many
SELECT * FROM link_visits
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: CountLinkVisits :one
SELECT count(*) FROM link_visits;

-- name: ListLinkVisitsByLinkID :many
SELECT * FROM link_visits
WHERE link_id = $1
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- name: CountLinkVisitsByLinkID :one
SELECT count(*) FROM link_visits
WHERE link_id = $1;
