-- name: CreateDepot :one
INSERT INTO depots (id, name, slug, phone, address)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDepot :one
SELECT * FROM depots WHERE id = $1;

-- name: GetDepotBySlug :one
SELECT * FROM depots WHERE slug = $1;

-- name: SlugExists :one
SELECT EXISTS (SELECT 1 FROM depots WHERE slug = $1);

-- name: UpdateDepot :one
UPDATE depots
SET name = COALESCE(sqlc.narg('name'), name),
    phone = COALESCE(sqlc.narg('phone'), phone),
    address = COALESCE(sqlc.narg('address'), address),
    lat = COALESCE(sqlc.narg('lat'), lat),
    lng = COALESCE(sqlc.narg('lng'), lng),
    open_time = COALESCE(sqlc.narg('open_time')::time, open_time),
    close_time = COALESCE(sqlc.narg('close_time')::time, close_time),
    delivery_fee = COALESCE(sqlc.narg('delivery_fee'), delivery_fee),
    is_accepting_orders = COALESCE(sqlc.narg('is_accepting_orders'), is_accepting_orders),
    auto_confirm_known = COALESCE(sqlc.narg('auto_confirm_known'), auto_confirm_known),
    loyalty_every = CASE WHEN sqlc.arg('clear_loyalty')::boolean THEN NULL ELSE COALESCE(sqlc.narg('loyalty_every'), loyalty_every) END,
    reminder_lead_days = COALESCE(sqlc.narg('reminder_lead_days'), reminder_lead_days),
    default_days_per_gallon = COALESCE(sqlc.narg('default_days_per_gallon'), default_days_per_gallon),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListDepots :many
SELECT * FROM depots ORDER BY created_at;

-- name: ListDemoDepots :many
SELECT * FROM depots WHERE is_demo ORDER BY created_at;
