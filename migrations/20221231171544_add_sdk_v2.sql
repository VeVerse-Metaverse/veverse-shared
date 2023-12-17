-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "sdk_v2"
(
    "id" uuid DEFAULT gen_random_uuid() NOT NULL
        CONSTRAINT sdk_v2_pk           -- primary key
            PRIMARY KEY
        CONSTRAINT sdk_v2_entity_id_fk -- reference to entity
            REFERENCES entities ("id")
            ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "sdk_v2";
-- +goose StatementEnd
