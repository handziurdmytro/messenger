package scylla

/*todo: migrations*/
const createKeyspace = `
    CREATE KEYSPACE IF NOT EXISTS messenger
    WITH replication = {
        'class': 'SimpleStrategy',
        'replication_factor': 1
    }
`

const createMessagesTable = `
    CREATE TABLE IF NOT EXISTS messenger.messages (
        room_id     text,
        created_at  timestamp,
        message_id  text,
        from_user   text,
        type        text,
        body        text,
        PRIMARY KEY ((room_id), created_at, message_id)
    ) WITH CLUSTERING ORDER BY (created_at DESC)
`
