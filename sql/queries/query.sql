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
WHERE email = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: RefreshTokenExpiry :one
SELECT expires_at 
FROM refresh_tokens
WHERE user_id = $1;

-- name: SaveRefToken :exec
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1, NOW(), NOW(), $2, NOW() + INTERVAL '60 days', NULL
);

-- name: GetUserFromRefreshToken :many
SELECT user_id, expires_at, revoked_at
FROM refresh_tokens
WHERE token = $1;

-- name: RevokeToken :exec
UPDATE refresh_tokens
SET updated_at = NOW(), revoked_at = NOW()
WHERE token = $1;

-- name: UpdateUser :exec
UPDATE users
SET email = $2, hashed_password = $3, updated_at = NOW()
WHERE id = $1;

-- name: UserFromToken :one
SELECT user_id
FROM refresh_tokens
WHERE token = $1;

-- name: GetUserEmail :one
SELECT email
FROM users
WHERE id = $1;