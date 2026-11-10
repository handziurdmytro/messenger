-- name: ApplyUserBlock :exec
INSERT INTO user_blocks (user_id, blocked_id, blocked, updated_at)
VALUES (sqlc.arg(user_id), sqlc.arg(blocked_id), sqlc.arg(blocked), sqlc.arg(updated_at))
ON CONFLICT (user_id, blocked_id) DO UPDATE
    SET blocked    = EXCLUDED.blocked,
        updated_at = EXCLUDED.updated_at
WHERE user_blocks.updated_at < EXCLUDED.updated_at;

-- name: BlockExistsBetween :one
SELECT EXISTS (SELECT 1
               FROM user_blocks
               WHERE blocked
                 AND ((user_id = sqlc.arg(first_id) AND blocked_id = sqlc.arg(second_id))
                   OR (user_id = sqlc.arg(second_id) AND blocked_id = sqlc.arg(first_id))));
