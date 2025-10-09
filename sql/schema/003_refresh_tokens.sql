-- +goose Up
CREATE TABLE refresh_tokens(
    token TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME
);

-- +goose Down
DROP TABLE refresh_tokens;