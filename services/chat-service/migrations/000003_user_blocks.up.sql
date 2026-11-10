CREATE TABLE user_blocks
(
    user_id    UUID        NOT NULL,
    blocked_id UUID        NOT NULL,
    blocked    BOOLEAN     NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (user_id, blocked_id)
);

CREATE INDEX idx_user_blocks_blocked_id
    ON user_blocks (blocked_id);
