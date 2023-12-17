-- +goose Up
-- +goose StatementBegin

create table if not exists entity_type_v2
(
    id          uuid primary key,
    name        text          not null,
    table_name  text          not null,
    version     int default 1 not null,
    description text,
    constraint entity_type_v2_name_table_name_key
        unique (name, table_name)
);

create index if not exists entity_type_v2_name_idx
    on entity_type_v2 (name);

create index if not exists entity_type_v2_table_name_idx
    on entity_type_v2 (table_name);

create index if not exists entity_type_v2_version_idx
    on entity_type_v2 (version);

comment on table entity_type_v2 is 'A type of entity. For example, a user, a world, a package, etc.';

insert into entity_type_v2 (id, name, table_name, version, description)
values ('407447B7-2A64-423B-8DC9-21A0FC5C4ECC', 'Launcher', 'launcher_v2', 2,
        'A launcher is a type of entity that represents a launcher that can have one or multiple linked apps.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('292BBCA9-78B0-4C11-8486-7F306D24931D', 'App', 'app_v2', 2,
        'A game or application. It is custom or VeVerse based (white-label) app that can have an SDK and can ' ||
        'be linked to a launcher. VeVerse based app can have multiple worlds, packages, and can have an SDK. ' ||
        'An app is owned by a user.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('AECBC3F3-FC6C-4EC6-AD69-BE9267BA9B68', 'Organization', 'organizations', 1,
        'An organization that can comprise multiple users.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('991545EE-8A98-4A64-AF51-9A04597BFE3C', 'User', 'users', 1,
        'A user identity, can own different types of entities, such as apps, worlds, packages, etc., can share ' ||
        'entities with other users, can have an organization. Identified with a username, a display name, ' ||
        'an email address, and optional blockchain wallet address.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('54484BB4-7B0D-4746-8B04-A375A34697E3', 'World', 'world_v2', 2,
        'A world is a type of entity that represents a virtual world that can be visited by users. A world is published to multiple apps by its owner.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('06A43CA1-2BA8-4102-BA7A-FEDBD8539195', 'Package', 'package_v2', 2,
        'A package is a type of entity that represents a package that can contain a world files or game mode files.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('2C677CC9-AED4-4457-8408-FA40A0696B7C', 'SDK', 'sdk_v2', 2,
        'An SDK is a type of entity that represents an SDK that is linked to a single or multiple supported apps.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('7B908E29-A702-44EE-BCEE-62928B190372', 'Asset Class', 'asset_class_v2', 2,
        'An asset class is a type of entity that represents a placeable asset class, such as a mesh, a media, an interactive object, an NPC, etc. Has a metadata field that describes what class to use in the game. Has a list of properties that can be used to define the asset class.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('F42BD3B5-C8E6-44B1-9359-915E0126355E', 'Asset', 'asset_v2', 2,
        'An asset is a type of entity that represents a placeable asset, each placeable asset has a asset class, placeable assets can be placed in worlds in the editor mode.');

insert into entity_type_v2 (id, name, table_name, version, description)
values ('C0B84BA2-C07D-42A8-A259-C5D172E0396A', 'Asset Bundle', 'asset_bundle_v2', 2,
        'An asset bundle is a type of entity that represents a collection of assets.');

create table if not exists entity_v2
(
    id             uuid                    not null
        primary key,
    entity_type_id uuid                    not null
        references entity_type_v2 (id),
    created_at     timestamp default now() not null,
    updated_at     timestamp,
    public         boolean   default false not null -- public means that the entity is visible to everyone
);

create index if not exists entity_v2_entity_type_id_idx
    on entity_v2 (entity_type_id);

comment on table entity_v2 is 'Entity is a thing that has traits, such as access metadata, tags, ratings, comments, etc.';

create table if not exists access_v2
(
    id         uuid primary key,
    entity_id  uuid                    not null
        references entity_v2 (id),
    user_id    uuid                    not null
        references users (id),
    created_at timestamp default now() not null,
    updated_at timestamp,
    is_owner   boolean   default false not null, -- is_owner means that the user owns the entity, and can do anything with it
    can_view   boolean   default false not null, -- can_view means that the user can view the entity
    can_edit   boolean   default false not null, -- can_edit means that the user can edit the entity
    can_delete boolean   default false not null, -- can_delete means that the user can delete the entity

    constraint access_entity_id_user_id_key
        unique (entity_id, user_id)
);

create index if not exists access_v2_entity_id_idx
    on access_v2 (entity_id);

create index if not exists access_v2_user_id_idx
    on access_v2 (user_id);

comment on table access_v2 is 'Access is a record of a user''s ownership or access to an entity.';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists access_v2;
drop table if exists entity_v2;
drop table if exists entity_type_v2;
-- +goose StatementEnd
