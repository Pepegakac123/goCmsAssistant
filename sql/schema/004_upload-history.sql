-- +goose Up
CREATE TABLE upload_history (
    id INTEGER PRIMARY KEY,
    filename TEXT NOT NULL,
    original_size INTEGER NOT NULL,
    webp_size INTEGER NOT NULL,
    wordpress_id INTEGER,
    wordpress_url TEXT,
    website_type TEXT NOT NULL,
    success INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE upload_history;