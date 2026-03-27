-- name: ListUsers :many
SELECT * FROM users ORDER BY name;

-- name: GetUser :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ? LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (name, username, password_hash) VALUES (?, ?, ?) RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name       = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: GetUserByAPIToken :one
SELECT * FROM users WHERE api_token = ? LIMIT 1;

-- name: SetUserAPIToken :exec
UPDATE users SET api_token = ? WHERE id = ?;
