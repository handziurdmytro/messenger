CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users
(
    id           TEXT PRIMARY KEY,
    email        TEXT        NOT NULL,
    display_name TEXT        NOT NULL,
    avatar_url   TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,

    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE TABLE contacts
(
    user_id    TEXT        NOT NULL,
    contact_id TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT contacts_pkey
        PRIMARY KEY (user_id, contact_id),

    CONSTRAINT contacts_user_id_fk
        FOREIGN KEY (user_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    CONSTRAINT contacts_contact_id_fk
        FOREIGN KEY (contact_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    CONSTRAINT contacts_not_self
        CHECK (user_id <> contact_id)
);

CREATE INDEX idx_users_display_name
    ON users USING gin (display_name gin_trgm_ops);