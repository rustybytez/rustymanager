-- name: ListChatMessagesByProject :many
SELECT cm.id, cm.project_id, cm.user_id, cm.content, cm.created_at,
       COALESCE(u.name, 'Anonymous') AS user_name
FROM chat_messages cm
LEFT JOIN users u ON cm.user_id = u.id
WHERE cm.project_id = ?
ORDER BY cm.created_at ASC
LIMIT 100;

-- name: CreateChatMessage :one
INSERT INTO chat_messages (project_id, user_id, content)
VALUES (?, ?, ?)
RETURNING *;
