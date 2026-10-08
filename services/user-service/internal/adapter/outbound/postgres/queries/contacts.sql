-- name: AddContact :exec
INSERT INTO contacts (user_id,
                      contact_id,
                      created_at)
VALUES (sqlc.arg(user_id),
        sqlc.arg(contact_id),
        sqlc.arg(created_at)),
       (sqlc.arg(contact_id),
        sqlc.arg(user_id),
        sqlc.arg(created_at));


-- name: GetContactsByUserID :many
SELECT u.id,
       u.email,
       u.display_name,
       u.avatar_url,
       u.created_at,
       u.updated_at
FROM users u
         INNER JOIN contacts c ON u.id = c.contact_id
WHERE c.user_id = sqlc.arg(user_id)
ORDER BY u.display_name ASC;