-- name: AddParticipant :exec
INSERT INTO participants (chat_id, user_id, role)
VALUES (sqlc.arg(chat_id), sqlc.arg(user_id), sqlc.arg(role));

-- name: ListParticipantsByChatID :many
SELECT user_id, role
FROM participants
WHERE chat_id = sqlc.arg(chat_id)
ORDER BY joined_at, user_id;

-- name: SetParticipantRole :exec
UPDATE participants
SET role = sqlc.arg(role)
WHERE chat_id = sqlc.arg(chat_id)
  AND user_id = sqlc.arg(user_id);

-- name: RemoveParticipant :exec
DELETE FROM participants
WHERE chat_id = sqlc.arg(chat_id)
  AND user_id = sqlc.arg(user_id);
