-- +goose Up
-- +goose StatementBegin

create table if not exists game_cloud_save
(
    id         uuid not null
        primary key
        references public.entities
            on delete cascade,
    created_at timestamp,
    updated_at timestamp,
    app_id     uuid not null, -- id of the app that created the save
    name       text not null  -- save name
);

comment on table game_cloud_save is 'Cloud save table (a save game stored in the cloud). Note: Uses accesibles table to track ownership. Uses files table to store the save file in the cloud.';

create index if not exists game_cloud_save_app_id_idx
    on game_cloud_save (app_id);

create index if not exists game_cloud_save_name_idx
    on game_cloud_save (name);

with e as (
    insert into entities (id, created_at, entity_type, public)
        values ('187D7D76-EA8B-4C2E-93AE-9576823548D1'::uuid, now(), 'game-cloud-save', true)
        returning id)
insert
into game_cloud_save (id, created_at, updated_at, app_id, name)
values ('187D7D76-EA8B-4C2E-93AE-9576823548D1'::uuid, now(), now(), '187D7D76-EA8B-4C2E-93AE-9576823548D1'::uuid,
        'default');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

delete
from entities
where id = '187D7D76-EA8B-4C2E-93AE-9576823548D1'::uuid;

drop table if exists game_cloud_save;

-- +goose StatementEnd
