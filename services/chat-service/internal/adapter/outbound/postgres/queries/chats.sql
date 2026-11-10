-- name: CreateChat :exec
INSERT INTO chats (id, type, title, private_key, created_at)
VALUES (sqlc.arg(id),
        sqlc.arg(type),
        sqlc.narg(title),
        sqlc.narg(private_key),
        sqlc.arg(created_at));

-- name: FindChatByID :one
SELECT id, type, title, private_key, created_at
FROM chats
WHERE id = sqlc.arg(id);

-- name: FindPrivateByParticipants :one
SELECT id, type, title, private_key, created_at
FROM chats
WHERE type = 'private'
  AND private_key = sqlc.arg(private_key);

-- name: ListChatsByUser :many
SELECT c.id,
       c.type,
       c.title,
       c.created_at,
       array_agg(p.user_id ORDER BY p.joined_at, p.user_id)::uuid[]                       AS participant_ids,
       coalesce(array_agg(p.user_id) FILTER (WHERE p.role = 'owner'), '{}')::uuid[] AS owner_ids
FROM chats c
         JOIN participants p ON p.chat_id = c.id
WHERE EXISTS (SELECT 1
              FROM participants mine
              WHERE mine.chat_id = c.id
                AND mine.user_id = sqlc.arg(user_id))
  AND (sqlc.narg(before_created_at)::timestamptz IS NULL
    OR (c.created_at, c.id) < (sqlc.narg(before_created_at)::timestamptz, sqlc.narg(before_id)::uuid))
GROUP BY c.id
ORDER BY c.created_at DESC, c.id DESC
LIMIT sqlc.arg(limit_count);

-- name: UpdateChatTitle :exec
UPDATE chats
SET title = sqlc.narg(title)
WHERE id = sqlc.arg(id);

-- name: DeleteChat :execrows
DELETE FROM chats
WHERE id = sqlc.arg(id);
