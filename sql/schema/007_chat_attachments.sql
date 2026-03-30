ALTER TABLE chat_messages ADD COLUMN attachment_url  TEXT NOT NULL DEFAULT '';
ALTER TABLE chat_messages ADD COLUMN attachment_type TEXT NOT NULL DEFAULT '';
