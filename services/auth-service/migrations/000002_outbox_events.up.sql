CREATE TABLE outbox_events
(
    id           text PRIMARY KEY,
    subject      text        NOT NULL,
    payload      bytea       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX outbox_events_pending_idx
    ON outbox_events (created_at, id)
    WHERE published_at IS NULL;