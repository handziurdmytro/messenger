ALTER TABLE chats
    ADD COLUMN title TEXT;

ALTER TABLE chats
    ADD CONSTRAINT chk_group_title
        CHECK (type <> 'group' OR title IS NOT NULL);

ALTER TABLE participants
    ADD COLUMN role TEXT NOT NULL DEFAULT 'member',
    ADD COLUMN joined_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp();

ALTER TABLE participants
    ADD CONSTRAINT chk_participant_role
        CHECK (role IN ('owner', 'member'));
