-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByName :one
SELECT * FROM users WHERE name = ?;

-- name: CreateUser :one
INSERT INTO users (name, role, hashed_password) 
VALUES (?, ?, ?) 
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserRoleById :one
SELECT role FROM users
WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;