-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS "launcher_apps_v2"
(
    "launcher_id" uuid    NOT NULL               -- reference to launcher
        CONSTRAINT "launcher_apps_v2_launcher_id_fkey"
            REFERENCES "launcher_v2" ("id")
            ON DELETE CASCADE,
    "app_id"      uuid    NOT NULL               -- reference to app
        CONSTRAINT "launcher_apps_v2_app_id_fkey"
            REFERENCES "app_v2" ("id")
            ON DELETE CASCADE,
    "published"   boolean NOT NULL DEFAULT false -- whether the app is published and visible to users within the launcher
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "launcher_apps_v2";
-- +goose StatementEnd
