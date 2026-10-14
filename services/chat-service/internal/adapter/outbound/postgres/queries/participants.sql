-- name: AddParticipant :exec
INSERT INTO participants (chat_id, user_id)
VALUES (sqlc.arg(chat_id), sqlc.arg(user_id));

-- name: ListParticipantsByChatID :many
SELECT user_id
FROM participants
WHERE chat_id = sqlc.arg(chat_id)
ORDER BY user_id;
