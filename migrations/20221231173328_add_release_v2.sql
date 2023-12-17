-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "release_v2"
(
    "id"              uuid          DEFAULT gen_random_uuid() NOT NULL
        CONSTRAINT release_v2_pk                              -- primary key
            PRIMARY KEY
        CONSTRAINT release_v2_entity_id_fk                    -- reference to entity
            REFERENCES entities ("id")
            ON DELETE CASCADE,
    "entity_id"       uuid NOT NULL
        CONSTRAINT release_v2_parent_entity_id_fk             -- reference to the parent entity (app, release, sdk, etc.)
            REFERENCES entities ("id")
            ON DELETE CASCADE,
    "version"         text NOT NULL DEFAULT '0.0.0',          -- release version (e.g. "1.0.0")
    "code_version"    text NOT NULL DEFAULT '0.0.0',          -- base Unreal Engine based client code version (e.g. "1.0.0")
    "content_version" text NOT NULL DEFAULT '1.0.0',          -- base Unreal Engine based client content version (e.g. "1.0.0")
    "archive"         bool NOT NULL DEFAULT false,            -- is this a release archive supplied as an archive file instead of list of separate files?
    "name"            text NOT NULL,                          -- release name visible to users (e.g. "Release 1.0.0")
    "description"     text          DEFAULT ''::text NOT NULL -- release description visible to users (e.g. "Release 1.0.0")
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "release_v2";
-- +goose StatementEnd
