-- name: CreateUploadHistory :one
INSERT INTO upload_history (
    filename, original_size, webp_size, wordpress_id, 
    wordpress_url, website_type, success, error_message, user_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUploadHistory :many
SELECT * FROM upload_history 
ORDER BY created_at DESC 
LIMIT ? OFFSET ?;

-- name: GetUploadHistoryByUser :many
SELECT * FROM upload_history 
WHERE user_id = ?
ORDER BY created_at DESC;