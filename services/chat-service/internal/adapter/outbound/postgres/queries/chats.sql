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
