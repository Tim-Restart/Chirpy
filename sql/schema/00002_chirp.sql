-- +goose Up
-- +goose StatementBegin
CREATE TABLE chirps(
    id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    body TEXT NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_users
    FOREIGN KEY (user_id)
    REFERENCES users(id)
    ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chirps;
-- +goose StatementEnd