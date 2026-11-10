-- name: CreateOutboxEvent :exec
INSERT INTO outbox_events (id, subject, payload)
VALUES (sqlc.arg(id), sqlc.arg(subject), sqlc.arg(payload));

-- name: ListPendingOutboxEvents :many
SELECT id, subject, payload
FROM outbox_events
WHERE published_at IS NULL
ORDER BY created_at, id
LIMIT sqlc.arg(max_events);

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET published_at = now()
WHERE id = sqlc.arg(id)
  AND published_at IS NULL;
