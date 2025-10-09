-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (name, role, hashed_password) VALUES ($1, $2, $3) RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;