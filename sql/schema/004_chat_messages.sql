CREATE TABLE IF NOT EXISTS chat_messages (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    INTEGER  REFERENCES users(id),
    content      TEXT     NOT NULL,
    message_type TEXT     NOT NULL DEFAULT 'message',
    room_name    TEXT     NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
