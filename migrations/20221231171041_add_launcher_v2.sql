-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "launcher_v2"
(
    "id"          uuid DEFAULT gen_random_uuid() NOT NULL
        CONSTRAINT launcher_v2_pk           -- primary key
            PRIMARY KEY
        CONSTRAINT launcher_v2_entity_id_fk -- reference to entity
            REFERENCES entities ("id")
            ON DELETE CASCADE,
    "name"        text                           NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "launcher_v2";
-- +goose StatementEnd
