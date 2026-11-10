-- name: AddBlock :exec
WITH removed_contacts AS (
    DELETE FROM contacts
    WHERE (user_id = sqlc.arg(user_id) AND contact_id = sqlc.arg(blocked_id))
       OR (user_id = sqlc.arg(blocked_id) AND contact_id = sqlc.arg(user_id)))
INSERT INTO blocks (user_id, blocked_id, created_at)
VALUES (sqlc.arg(user_id), sqlc.arg(blocked_id), sqlc.arg(created_at));

-- name: RemoveBlock :execrows
DELETE FROM blocks
WHERE user_id = sqlc.arg(user_id)
  AND blocked_id = sqlc.arg(blocked_id);

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
