create schema public;

comment on schema public is 'standard public schema';

alter schema public owner to postgres;

create table "group"
(
    id uuid not null,
    parent_id uuid not null,
    name bytea not null
        constraint group_unique_name
            unique,
    flags integer default 0 not null,
    key bytea not null,
    constraint group_pk
        unique (id, parent_id)
);

alter table "group" owner to lazyfingers;

create unique index group_id_uindex
    on "group" (id);

create unique index group_key_uindex
    on "group" (key);

create index group_flags_index
    on "group" (flags);

create table group_assets
(
    group_id uuid not null
        constraint group_assets_group_id_fk
            references "group" (id)
            on delete cascade,
    asset_id uuid not null,
    asset_kind smallint default 0 not null,
    constraint group_assets_pk
        primary key (group_id, asset_id, asset_kind)
);

alter table group_assets owner to lazyfingers;

create index group_assets_asset_id_index
    on group_assets (asset_id);

create index group_assets_group_id_index
    on group_assets (group_id);

create table accesspolicy
(
    id uuid not null
        constraint accesspolicy_id_pk
            primary key,
    parent_id uuid,
    owner_id uuid not null,
    key bytea
        constraint accesspolicy_pk
            unique,
    object_name bytea,
    object_id uuid,
    flags smallint default 0 not null,
    constraint accesspolicy_pk_object_name_id
        unique (object_name, object_id)
);

alter table accesspolicy owner to lazyfingers;

create unique index accesspolicy__key_uindex
    on accesspolicy (key)
    where (key IS NOT NULL);

create table accesspolicy_roster
(
    policy_id uuid not null,
    actor_kind smallint not null,
    actor_id uuid not null,
    access bigint not null,
    access_explained text,
    constraint accesspolicy_roster_pk
        primary key (policy_id, actor_kind, actor_id)
);

alter table accesspolicy_roster owner to lazyfingers;

create index accesspolicy_roster_policy_id_actor_kind_index
    on accesspolicy_roster (policy_id, actor_kind);

create index accesspolicy_roster_policy_id_index
    on accesspolicy_roster (policy_id);

create table password
(
    kind smallint not null,
    owner_id uuid not null,
    hash bytea not null,
    is_change_required boolean default false not null,
    created_at timestamp not null,
    updated_at timestamp,
    expire_at timestamp,
    constraint password_pk
        primary key (kind, owner_id)
);

alter table password owner to lazyfingers;

create table user_email
(
    user_id uuid not null,
    addr bytea not null
        constraint user_email_pk
            primary key,
    is_primary boolean default false not null,
    created_at timestamp not null,
    confirmed_at timestamp,
    updated_at timestamp
);

alter table user_email owner to lazyfingers;

create index user_email_user_id_index
    on user_email (user_id);

create table user_phone
(
    user_id uuid not null,
    number bytea,
    is_primary boolean default false,
    created_at timestamp not null,
    confirmed_at timestamp,
    updated_at timestamp
);

alter table user_phone owner to lazyfingers;

create unique index user_phone_number_uindex
    on user_phone (number);

create index user_phone_user_id_index
    on user_phone (user_id);

create table user_profile
(
    user_id uuid not null
        constraint user_profile_pk
            primary key,
    firstname bytea,
    middlename bytea,
    lastname bytea,
    language bytea,
    checksum bigint default 0 not null,
    created_at timestamp not null,
    updated_at timestamp
);

alter table user_profile owner to lazyfingers;

create index user_profile_created_at_index
    on user_profile (created_at);

create index user_profile_firstname_index
    on user_profile (firstname);

create index user_profile_language_index
    on user_profile (language);

create index user_profile_lastname_index
    on user_profile (lastname);

create index user_profile_middlename_index
    on user_profile (middlename);

create index user_profile_updated_at_index
    on user_profile (updated_at);

create index user_profile_user_id_index
    on user_profile (user_id);

create table token
(
    kind smallint not null,
    hash bytea not null
        constraint token_pk
            primary key,
    checkin_total integer not null,
    checkin_remainder integer not null,
    created_at timestamp not null,
    expire_at timestamp
);

alter table token owner to lazyfingers;

create index token_created_at_index
    on token (created_at);

create index token_expire_at_index
    on token (expire_at);

create table "user"
(
    id uuid not null
        constraint user_pk
            primary key,
    username bytea not null,
    display_name bytea not null,
    last_login_at timestamp,
    last_login_ip inet,
    last_login_failed_at timestamp,
    last_login_failed_ip inet,
    last_login_attempts smallint not null,
    is_suspended boolean default false,
    suspension_reason text,
    suspension_expires_at timestamp,
    suspended_by_id uuid,
    checksum bigint,
    confirmed_at timestamp,
    created_at timestamp,
    created_by_id uuid,
    updated_at timestamp,
    updated_by_id uuid,
    deleted_at timestamp,
    deleted_by_id uuid
);

alter table "user" owner to lazyfingers;

create unique index user_display_name_uindex
    on "user" (display_name);

create unique index user_username_uindex
    on "user" (username);

