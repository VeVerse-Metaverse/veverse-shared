-- +goose Up
-- +goose StatementBegin

create table if not exists game_lobby
(
    id          uuid not null
        primary key
        references public.entities
            on delete cascade,
    created_at  timestamp,
    updated_at  timestamp, -- lobby last update time (status change)
    server_id   uuid       -- id of the server the lobby is waiting to join (server is created when the lobby is ready)
        references game_server_v2
            on delete cascade,
    max_players int,       -- max players allowed in the lobby
    status      text       -- lobby status (waiting - waiting for players to join, ready - ready to join the server, failed - failed to join the server, closed - lobby closed)
);

comment on table game_lobby is 'Lobby table (lobby is a group of players waiting to join a game server). Note: Uses properties table to store lobby custom properties. Uses accesibles table to track ownership.';

with e as (
    insert into public.entities (id, entity_type, created_at, updated_at)
        values ('9419BB8E-D910-470A-9E59-3BA2D8134970'::uuid, 'game-lobby', now(), now())
        returning id)
insert
into game_lobby (id, created_at, updated_at, server_id, max_players, status)
select e.id, now(), now(), '83CCADBA-B3BF-494A-9FD5-B4F35777CF78'::uuid, 10, 'closed'
from e;

create table if not exists game_lobby_player
(
    lobby_id   uuid not null -- lobby id
        references game_lobby
            on delete cascade,
    user_id    uuid not null -- player who joined the lobby
        references public.entities
            on delete cascade,
    created_at timestamp,    -- player connection time
    updated_at timestamp,    -- player last update time ( status change)
    status     text          -- player status (joined - player joined the lobby, ready - player is ready to join the server, failed - player failed to join the server, left - player left the lobby)
);

comment on table game_lobby_player is 'Lobby player table (lobby player is a player waiting to join a game server).';

create index if not exists game_lobby_player_lobby_id_idx
    on game_lobby_player (lobby_id);

create index if not exists game_lobby_player_user_id_idx
    on game_lobby_player (user_id);

create index if not exists game_lobby_player_status_idx
    on game_lobby_player (status);

insert into game_lobby_player (lobby_id, user_id, created_at, updated_at, status)
values ('9419BB8E-D910-470A-9E59-3BA2D8134970'::uuid, '1578BA66-3334-496E-8BB8-1A0696B42C68'::uuid, now(), now(),
        'left');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

delete
from entities
where id = '9419BB8E-D910-470A-9E59-3BA2D8134970'::uuid;

drop table if exists game_lobby_player;
drop table if exists game_lobby;

-- +goose StatementEnd
