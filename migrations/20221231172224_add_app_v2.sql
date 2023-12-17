-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "app_v2"
(
    "id"          uuid    DEFAULT gen_random_uuid() NOT NULL
        CONSTRAINT app_v2_pk           -- primary key
            PRIMARY KEY
        CONSTRAINT app_v2_entity_id_fk -- reference to entity
            REFERENCES entities ("id")
            ON DELETE CASCADE,
    "name"        text                              NOT NULL,
    "description" text    DEFAULT ''::text          NOT NULL,
    "external"    boolean DEFAULT false             NOT NULL,
    "sdk_id"      uuid    DEFAULT NULL -- reference to sdk
        CONSTRAINT app_v2_sdk_id_fk
            REFERENCES "sdk_v2" ("id")
            ON DELETE SET NULL         -- allow sdk to be deleted
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "app_v2";
-- +goose StatementEnd
