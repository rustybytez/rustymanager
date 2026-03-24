-- name: ListUsers :many
SELECT * FROM users ORDER BY name;

-- name: GetUser :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (name) VALUES (?) RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name       = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;
