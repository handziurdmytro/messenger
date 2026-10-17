CREATE TABLE IF NOT EXISTS messenger.messages_by_chat
(
    chat_id    text,
    bucket     int,
    created_at timestamp,
    message_id uuid,
    from_user  text,
    type       text,
    payload    text,
    edited_at  timestamp,
    deleted_at timestamp,

    PRIMARY KEY ((chat_id, bucket), created_at, message_id)
) WITH CLUSTERING ORDER BY (
                            created_at DESC,
                            message_id DESC
    );
