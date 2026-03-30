-- name: ListChatMessagesByProject :many
SELECT cm.id, cm.project_id, cm.user_id, cm.content, cm.message_type, cm.room_name,
       cm.attachment_url, cm.attachment_type, cm.created_at,
       COALESCE(u.name, 'Anonymous') AS user_name
FROM chat_messages cm
LEFT JOIN users u ON cm.user_id = u.id
WHERE cm.project_id = ?
ORDER BY cm.created_at DESC
LIMIT 100;

-- name: ListChatMessagesBefore :many
SELECT cm.id, cm.project_id, cm.user_id, cm.content, cm.message_type, cm.room_name,
       cm.attachment_url, cm.attachment_type, cm.created_at,
       COALESCE(u.name, 'Anonymous') AS user_name
FROM chat_messages cm
LEFT JOIN users u ON cm.user_id = u.id
WHERE cm.project_id = ? AND cm.id < ?
ORDER BY cm.created_at DESC
LIMIT 50;

-- name: CreateChatMessage :one
INSERT INTO chat_messages (project_id, user_id, content, message_type, room_name, attachment_url, attachment_type)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetActiveCallForProject :one
SELECT room_name, message_type FROM chat_messages
WHERE project_id = ? AND message_type IN ('call_start', 'call_end')
ORDER BY id DESC
LIMIT 1;
