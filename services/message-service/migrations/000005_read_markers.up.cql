CREATE TABLE IF NOT EXISTS messenger_messages.read_markers
(
    chat_id text,
    user_id text,
    message_id uuid,
    read_at timestamp,
    PRIMARY KEY ((chat_id), user_id)
);
