-- +goose Up
-- +goose StatementBegin
CREATE TABLE users(
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	email TEXT NOT NULL UNIQUE
);
-- +goose StatementEnd
ALTER TABLE users
ADD COLUMN hashed_password TEXT NOT NULL;

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
