-- name: CreateEmailVerification :one
INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetEmailByToken :one
SELECT * FROM email_verification_tokens
WHERE token_hash = $1 AND expires_at > now(); 

-- name: MarkEmailVerified :exec
UPDATE users
SET email_verified = true
WHERE primary_email = $1;
