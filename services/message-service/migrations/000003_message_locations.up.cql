CREATE TABLE IF NOT EXISTS messenger.message_locations
(
    chat_id      text,
    message_id   uuid,
    bucket       int,
    created_at   timestamp,
    request_hash text,

    PRIMARY KEY ((chat_id), message_id)
);
