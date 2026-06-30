CREATE TABLE email_verification_tokens (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
token_hash TEXT NOT NULL,
expires_at TIMESTAMP NOT NULL,
created_at TIMESTAMP NOT NULL DEFAULT now(),

UNIQUE(user_id)
);

ALTER TABLE users
ADD email_verified BOOLEAN NOT NULL DEFAULT false;
