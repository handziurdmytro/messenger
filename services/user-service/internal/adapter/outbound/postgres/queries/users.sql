-- name: CreateUser :execrows
INSERT INTO users (id, email, display_name, avatar_url, created_at, updated_at)
VALUES (sqlc.arg(id),
        sqlc.arg(email),
        sqlc.arg(display_name),
        sqlc.arg(avatar_url),
        sqlc.arg(created_at),
        sqlc.arg(updated_at))
ON CONFLICT (id) DO UPDATE
    SET id = EXCLUDED.id
WHERE users.email = EXCLUDED.email
  AND users.created_at = EXCLUDED.created_at;

-- name: GetUserByID :one
SELECT id,
       email,
       display_name,
       avatar_url,
       created_at,
       updated_at
FROM users
WHERE id = sqlc.arg(id);

-- name: GetUsersByIDs :many
SELECT id,
       email,
       display_name,
       avatar_url,
       created_at,
       updated_at
FROM users
WHERE id = ANY (sqlc.arg(ids)::text[]);

-- name: UpdateUser :execrows
UPDATE
    users
SET display_name = sqlc.arg(display_name),
    avatar_url   = sqlc.arg(avatar_url),
    updated_at   = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: SearchUsers :many
SELECT id,
       email,
       display_name,
       avatar_url,
       created_at,
       updated_at
FROM users
WHERE display_name ILIKE '%' || sqlc.arg(query)::text || '%'
   OR email ILIKE '%' || sqlc.arg(query)::text || '%'
LIMIT sqlc.arg(limit_count);

