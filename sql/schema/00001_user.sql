-- +goose Up
CREATE TABLE users(
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	email TEXT NOT NULL UNIQUE
);
ALTER TABLE users
ADD COLUMN hashed_password TEXT NOT NULL;

ALTER TABLE users
ADD COLUMN is_chirpy_red BOOLEAN DEFAULT false;

-- +goose Down
DROP TABLE users;

