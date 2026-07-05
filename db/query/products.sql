-- Stock management queries for comment-to-order (ADR-001 §1.5).
-- DecrementStock and IncrementStock are single-UPDATE atomic guards —
-- the WHERE condition prevents oversell and double-release (§10 concurrency).

-- name: GetProductByPostAndCode :one
SELECT * FROM product
WHERE catalog_post_id = @catalog_post_id
  AND code            = @code;

-- name: UpsertProduct :one
-- Idempotent registration of a keep/C product (used by cmd/seed). On
-- conflict only identity fields (name/price) are refreshed — stock is left
-- untouched so re-seeding doesn't reset stock consumed by prior test runs.
INSERT INTO product (catalog_post_id, code, name, price_idr, stock_total, stock_left)
VALUES (@catalog_post_id, @code, @name, @price_idr, @stock_total, @stock_total)
ON CONFLICT (catalog_post_id, code) DO UPDATE SET
    name      = EXCLUDED.name,
    price_idr = EXCLUDED.price_idr
RETURNING *;

-- name: DecrementStock :one
-- Atomically claim one unit. Returns the updated row; returns no rows if stock_left = 0.
-- Caller treats zero rows as "out of stock" — do NOT create a reservation in that case.
UPDATE product
SET stock_left = stock_left - 1
WHERE id         = @id
  AND stock_left > 0
RETURNING *;

-- name: IncrementStock :one
-- Atomically release one unit (auto-release on expire). Guards against exceeding
-- stock_total via LEAST so a double-release still yields a consistent state.
UPDATE product
SET stock_left = LEAST(stock_left + 1, stock_total)
WHERE id = @id
RETURNING *;
