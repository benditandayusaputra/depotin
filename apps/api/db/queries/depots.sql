-- name: CreateDepot :one
INSERT INTO depots (id, name, slug, phone, address)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDepot :one
SELECT * FROM depots WHERE id = $1;

-- name: GetDepotBySlug :one
SELECT * FROM depots WHERE slug = $1;
