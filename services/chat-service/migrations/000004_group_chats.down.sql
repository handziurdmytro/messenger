DELETE FROM chats
WHERE type = 'group';

ALTER TABLE participants
    DROP CONSTRAINT chk_participant_role,
    DROP COLUMN joined_at,
    DROP COLUMN role;

ALTER TABLE chats
    DROP CONSTRAINT chk_group_title,
    DROP COLUMN title;
