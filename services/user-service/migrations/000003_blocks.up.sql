CREATE TABLE blocks
(
    user_id    TEXT        NOT NULL,
    blocked_id TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT blocks_pkey
        PRIMARY KEY (user_id, blocked_id),

    CONSTRAINT blocks_user_id_fk
        FOREIGN KEY (user_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    CONSTRAINT blocks_blocked_id_fk
        FOREIGN KEY (blocked_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    CONSTRAINT blocks_not_self
        CHECK (user_id <> blocked_id)
);

CREATE INDEX idx_blocks_blocked_id
    ON blocks (blocked_id);
