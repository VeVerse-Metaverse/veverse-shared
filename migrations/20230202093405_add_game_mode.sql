-- +goose Up
-- +goose StatementBegin

create table if not exists game_mode
(
    id   uuid not null
        primary key
        references entities (id)
            on delete cascade,
    name text, -- display name
    path text  -- game mode blueprint path (relative to the game server files)
);

comment on table game_mode is 'Game mode table (game mode is a game mode blueprint like TDM, CTF, etc).';

create index if not exists game_mode_name_idx
    on game_mode (name);


with e as (
    insert into entities (id, created_at, entity_type, public)
        values ('8835AE80-7F42-4CE1-B695-7716CF9530A8', now(), 'game-mode', true)
        returning id)
insert
into game_mode (id, name, path)
select e.id, 'Example', 'Game/Blueprints/GameModes/Example/BP_ExampleGameMode.BP_ExampleGameMode_C'
from e;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

delete
from entity_v2
where id = '8835AE80-7F42-4CE1-B695-7716CF9530A8';

drop table if exists game_mode;

-- +goose StatementEnd
