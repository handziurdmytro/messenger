DELETE FROM users
WHERE deleted_at IS NOT NULL;

DROP INDEX users_email_unique;

ALTER TABLE users
    ADD CONSTRAINT users_email_unique UNIQUE (email);

ALTER TABLE users
    DROP COLUMN deleted_at;
