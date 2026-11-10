-- name: AddBlock :exec
WITH removed_contacts AS (
    DELETE FROM contacts
    WHERE (contacts.user_id = sqlc.arg(user_id) AND contacts.contact_id = sqlc.arg(blocked_id))
       OR (contacts.user_id = sqlc.arg(blocked_id) AND contacts.contact_id = sqlc.arg(user_id))),
     inserted AS (
         INSERT INTO blocks (user_id, blocked_id, created_at)
         VALUES (sqlc.arg(user_id), sqlc.arg(blocked_id), sqlc.arg(created_at))
         RETURNING blocks.user_id)
INSERT INTO outbox_events (id, subject, payload)
SELECT sqlc.arg(event_id), sqlc.arg(event_subject), sqlc.arg(event_payload)::bytea
FROM inserted;

-- name: RemoveBlock :execrows
WITH removed AS (
    DELETE FROM blocks
    WHERE blocks.user_id = sqlc.arg(user_id)
      AND blocks.blocked_id = sqlc.arg(blocked_id)
    RETURNING blocks.user_id)
INSERT INTO outbox_events (id, subject, payload)
SELECT sqlc.arg(event_id), sqlc.arg(event_subject), sqlc.arg(event_payload)::bytea
FROM removed;

-- name: ListBlockedUsers :many
SELECT u.id,
       u.email,
       u.display_name,
       u.avatar_url,
       u.created_at,
       u.updated_at,
       u.deleted_at
FROM users u
         INNER JOIN blocks b ON u.id = b.blocked_id
WHERE b.user_id = sqlc.arg(user_id)
ORDER BY b.created_at DESC, u.id
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: BlockExistsBetween :one
SELECT EXISTS (SELECT 1
               FROM blocks
               WHERE (user_id = sqlc.arg(first_id) AND blocked_id = sqlc.arg(second_id))
                  OR (user_id = sqlc.arg(second_id) AND blocked_id = sqlc.arg(first_id)));
