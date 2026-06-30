-- name: CreateUser :one
INSERT INTO users (name, primary_email) 
    VALUES ($1, $2)
RETURNING *;

-- name: CreateAccount :one
INSERT INTO accounts (user_id, provider, access_token, password_hash) 
VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetUserById :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE primary_email = $1;

-- name: GetUserAccounts :many
SELECT * FROM accounts
WHERE user_id = $1;

