CREATE TABLE IF NOT EXISTS projects (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    name        TEXT     NOT NULL,
    description TEXT     NOT NULL DEFAULT '',
    status      TEXT     NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active', 'archived')),
    created_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    created_by INTEGER  REFERENCES users(id),
    updated_by INTEGER  REFERENCES users(id)
);

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

ALTER TABLE projects ADD COLUMN created_by INTEGER REFERENCES users(id);
ALTER TABLE projects ADD COLUMN updated_by INTEGER REFERENCES users(id);

ALTER TABLE projects ADD COLUMN github_repo TEXT NOT NULL DEFAULT '';
ALTER TABLE kanban_items ADD COLUMN deleted_at DATETIME;

CREATE TABLE IF NOT EXISTS chat_messages (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER  NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    INTEGER  REFERENCES users(id),
    content    TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    endpoint   TEXT     NOT NULL UNIQUE,
    p256dh     TEXT     NOT NULL,
    auth       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
