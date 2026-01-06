-- name: CreateSubscription :one
INSERT INTO subscriptions (name, amount, currency, billing_cycle, next_renewal_date)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSubscription :one
SELECT * FROM subscriptions WHERE id = ?;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions ORDER BY name ASC;

-- name: ListSubscriptionsByBillingCycle :many
SELECT * FROM subscriptions WHERE billing_cycle = ? ORDER BY name ASC;

-- name: ListMonthlySubscriptions :many
SELECT * FROM subscriptions WHERE billing_cycle = 'monthly' ORDER BY name ASC;

-- name: ListYearlySubscriptions :many
SELECT * FROM subscriptions WHERE billing_cycle = 'yearly' ORDER BY next_renewal_date ASC;

-- name: UpdateSubscription :one
UPDATE subscriptions
SET name = ?, amount = ?, currency = ?, billing_cycle = ?, next_renewal_date = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: UpdateRenewalDate :one
UPDATE subscriptions
SET next_renewal_date = ?, updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteSubscription :exec
DELETE FROM subscriptions WHERE id = ?;

-- name: GetYearlySubscriptionsRenewingInMonth :many
SELECT * FROM subscriptions
WHERE billing_cycle = 'yearly' AND strftime('%Y-%m', next_renewal_date) = ?
ORDER BY next_renewal_date ASC;

-- name: GetAllSubscriptionsForExport :many
SELECT id, name, amount, currency, billing_cycle, next_renewal_date, created_at, updated_at
FROM subscriptions
ORDER BY name ASC;

-- Config queries
-- name: GetConfig :one
SELECT value FROM config WHERE key = ?;

-- name: SetConfig :exec
INSERT INTO config (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: GetAllConfig :many
SELECT key, value FROM config ORDER BY key;
