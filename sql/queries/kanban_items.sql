-- name: ListKanbanItemsByProject :many
SELECT
    ki.id,
    ki.project_id,
    ki.title,
    ki.assignee_id,
    ki.status,
    ki.created_at,
    ki.updated_at,
    ki.created_by,
    ki.updated_by,
    u.name AS assignee_name
FROM kanban_items ki
LEFT JOIN users u ON u.id = ki.assignee_id
WHERE ki.project_id = ?
ORDER BY ki.status, ki.created_at;

-- name: GetKanbanItem :one
SELECT * FROM kanban_items WHERE id = ? LIMIT 1;

-- name: CreateKanbanItem :one
INSERT INTO kanban_items (project_id, title, assignee_id, status)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: UpdateKanbanItem :one
UPDATE kanban_items
SET title       = ?,
    assignee_id = ?,
    status      = ?,
    updated_at  = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: UpdateKanbanItemStatus :one
UPDATE kanban_items
SET status     = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ?
RETURNING *;

-- name: DeleteKanbanItem :exec
DELETE FROM kanban_items WHERE id = ?;
