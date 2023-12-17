-- +goose Up
-- +goose StatementBegin

create table if not exists game_server_v2
(
    id             uuid not null
        primary key
        references public.entities
            on delete cascade,
    created_at     timestamp,        -- server creation time (the server is created by the API or by the orchestrator using scheduler or during scaling)
    updated_at     timestamp,        -- server last update time (heartbeat or status change)
    release_id     uuid              -- release id (to download the server files to the pod)
                        references release_v2
                            on delete set null,
    world_id       uuid default null -- id of the world running on this server
                        references spaces
                            on delete set null,
    game_mode_id   uuid default null -- id of the game class running on this server
                        references game_mode
                            on delete set null,
    type           text,             -- server type (official, community)
    region         text,             -- server region, in cluster deployments this is the cluster region (e.g. us-east-1), in community deployments can be any string or empty
    host           text,             -- server host, in cluster deployments this is the cluster service host, in community deployments this is the server public IP
    port           int,              -- server port, in cluster deployments this is the cluster service port, in community deployments this is the server public port
    max_players    int,              -- server max players allowed at the server
    status         text,             -- server status, in cluster deployments this is the cluster pod status (created, deploying, online, offline, failed), in community deployments this is the server status (online, offline, etc).
    status_message text default null -- server status message (error message or empty)
);

comment on table game_server_v2 is 'Game server table (game server is a game instance hosting a game world with a game class).';

create index if not exists game_server_release_id_idx
    on game_server_v2 (release_id);

create index if not exists game_server_world_id_idx
    on game_server_v2 (world_id);

create index if not exists game_server_game_mode_id_idx
    on game_server_v2 (game_mode_id);

create index if not exists game_server_type_idx
    on game_server_v2 (type);

create index if not exists game_server_region_idx
    on game_server_v2 (region);

create index if not exists game_server_status_idx
    on game_server_v2 (status);

create index if not exists game_server_updated_at_idx
    on game_server_v2 (updated_at);

create table if not exists game_server_player_v2
(
    server_id  uuid not null
        references game_server_v2
            on delete cascade,
    user_id    uuid not null
        references public.entities
            on delete cascade,
    created_at timestamp, -- player connection time
    updated_at timestamp, -- player last update time (heartbeat or status change)
    status     text       -- player status (connected, disconnected)
);

comment on table game_server_player_v2 is 'Game server player table (game server player is a player connected to a game server).';

create index if not exists game_server_player_server_id_idx
    on game_server_player_v2 (server_id);

create index if not exists game_server_player_user_id_idx
    on game_server_player_v2 (user_id);

create index if not exists game_server_player_status_idx
    on game_server_player_v2 (status);

create index if not exists game_server_player_updated_at_idx
    on game_server_player_v2 (updated_at);

with e as (
    insert into entities (id, created_at, entity_type, public)
        values ('83CCADBA-B3BF-494A-9FD5-B4F35777CF78'::uuid, now(), 'game-server-v2', true)
        returning id)
insert
into game_server_v2 (id, created_at, updated_at, release_id, game_mode_id, world_id, type, region, host, port,
                     max_players, status, status_message)
select e.id,
       now(),
       now(),
       '6243C871-4890-418F-8E60-00FAFDD48907'::uuid,
       '8835AE80-7F42-4CE1-B695-7716CF9530A8'::uuid,
       'D15007F4-CF4E-4069-88DE-17307961204D'::uuid,
       'official',
       'aws-region-xx',
       'xx.xxxx.veverse.com',
       7777,
       100,
       'offline',
       null
from e;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

delete
from entities
where id = '83CCADBA-B3BF-494A-9FD5-B4F35777CF78'::uuid;

drop table if exists game_server_player_v2;
drop table if exists game_server_v2;

-- +goose StatementEnd
