CREATE KEYSPACE IF NOT EXISTS messenger_messages
WITH replication = {'class': 'NetworkTopologyStrategy', 'datacenter1': 1}
AND durable_writes = true
AND tablets = {'enabled': false};

CREATE TABLE IF NOT EXISTS messenger_messages.messages_by_chat
(
    chat_id text,
    bucket int,
    created_at timestamp,
    message_id uuid,
    from_user text,
    type text,
    payload text,
    edited_at timestamp,
    deleted_at timestamp,
    PRIMARY KEY ((chat_id, bucket), created_at, message_id)
) WITH CLUSTERING ORDER BY (created_at DESC, message_id DESC);

CREATE TABLE IF NOT EXISTS messenger_messages.message_locations
(
    chat_id text,
    message_id uuid,
    bucket int,
    created_at timestamp,
    request_hash text,
    PRIMARY KEY ((chat_id), message_id)
);
