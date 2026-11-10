ALTER TABLE users
    ADD COLUMN deleted_at TIMESTAMPTZ;

ALTER TABLE users
    DROP CONSTRAINT users_email_unique;

CREATE UNIQUE INDEX users_email_unique
    ON users (email)
    WHERE deleted_at IS NULL;
