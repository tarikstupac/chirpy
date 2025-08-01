-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: RetrieveUserByEmail :one
SELECT *
FROM users
WHERE email = $1
LIMIT 1;

-- name: RetrieveUserByRefreshToken :one
SELECT u.*
FROM users u
JOIN refresh_tokens rt ON u.id = rt.user_id
WHERE rt.token = $1 AND rt.revoked_at IS NULL
LIMIT 1;

-- name: UpdateUserEmailAndPassword :one
UPDATE users
SET email = $1,
    hashed_password = $2,
    updated_at = NOW()
WHERE id = $3
RETURNING *;

-- name: UpdateUserChirpyRedStatus :exec
UPDATE users
SET is_chirpy_red = $1,
    updated_at = NOW()
WHERE id = $2;

-- name: RetrieveUserById :one
SELECT *
FROM users
WHERE id = $1
LIMIT 1;