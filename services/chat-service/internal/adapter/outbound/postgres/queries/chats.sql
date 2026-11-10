-- name: CreateChat :exec
INSERT INTO chats (id, type, private_key, created_at)
VALUES (sqlc.arg(id),
        sqlc.arg(type),
        sqlc.arg(private_key),
        sqlc.arg(created_at));

-- name: FindChatByID :one
SELECT id, type, private_key, created_at
FROM chats
WHERE id = sqlc.arg(id);

-- name: FindPrivateByParticipants :one
SELECT id, type, private_key, created_at
FROM chats
WHERE type = 'private'
  AND private_key = sqlc.arg(private_key);

-- name: ListChatsByUser :many
SELECT c.id,
       c.type,
       c.created_at,
       array_agg(p.user_id ORDER BY p.user_id)::uuid[] AS participant_ids
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
