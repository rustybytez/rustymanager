-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC;

-- name: GetProject :one
SELECT * FROM projects WHERE id = ? LIMIT 1;

-- name: CreateProject :one
INSERT INTO projects (name, description, status, github_repo) VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name        = ?,
    description = ?,
    status      = ?,
    github_repo = ?,
    updated_at  = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = ?;
