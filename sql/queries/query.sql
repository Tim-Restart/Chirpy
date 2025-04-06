-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: NewChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: GetChirps :many
SELECT * FROM chirps 
ORDER BY created_at ASC;

-- name: GetChirp :one
SELECT * 
FROM chirps
WHERE ID = $1;

-- name: GetEmail :one
SELECT *
FROM users
WHERE ID = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;