CREATE TABLE IF NOT EXISTS users (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    name          TEXT     NOT NULL,
    username      TEXT     NOT NULL DEFAULT '',
    password_hash TEXT     NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at    DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    created_by    INTEGER  REFERENCES users(id),
    updated_by    INTEGER  REFERENCES users(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS users_username_idx ON users(username) WHERE username != '';
