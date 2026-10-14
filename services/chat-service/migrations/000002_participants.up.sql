CREATE TABLE participants
(
    chat_id UUID NOT NULL,
    user_id UUID NOT NULL,

    PRIMARY KEY (chat_id, user_id),
    CONSTRAINT fk_participants_chat
        FOREIGN KEY (chat_id)
            REFERENCES chats (id)
            ON DELETE CASCADE
);

CREATE INDEX idx_participants_user_id
    ON participants (user_id);