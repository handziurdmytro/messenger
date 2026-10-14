CREATE TABLE chats
(
    id          UUID PRIMARY KEY,
    type        TEXT        NOT NULL,
    private_key TEXT,
    created_at  TIMESTAMPTZ NOT NULL,

    CONSTRAINT chk_chat_type
        CHECK (type IN ('private', 'group')),

    CONSTRAINT uq_chats_private_key
        UNIQUE (private_key),

    CONSTRAINT chk_private_key CHECK (
        (type = 'private' AND private_key IS NOT NULL)
            OR
        (type = 'group' AND private_key IS NULL)
    )
);