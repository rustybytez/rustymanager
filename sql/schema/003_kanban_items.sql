CREATE TABLE IF NOT EXISTS kanban_items (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    project_id  INTEGER  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title       TEXT     NOT NULL,
    assignee_id INTEGER  REFERENCES users(id),
    status      TEXT     NOT NULL DEFAULT 'todo'
                         CHECK (status IN ('todo', 'in_progress', 'done')),
    created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    created_by  INTEGER  REFERENCES users(id),
    updated_by  INTEGER  REFERENCES users(id)
);
