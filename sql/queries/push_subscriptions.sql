-- name: UpsertPushSubscription :exec
INSERT INTO push_subscriptions (endpoint, p256dh, auth, user_id)
VALUES (?, ?, ?, ?)
ON CONFLICT(endpoint) DO UPDATE SET p256dh = excluded.p256dh, auth = excluded.auth, user_id = excluded.user_id;

-- name: ListPushSubscriptionsExcludingUser :many
SELECT endpoint, p256dh, auth FROM push_subscriptions
WHERE user_id IS NULL OR user_id != ?;

-- name: DeletePushSubscription :exec
DELETE FROM push_subscriptions WHERE endpoint = ?;
