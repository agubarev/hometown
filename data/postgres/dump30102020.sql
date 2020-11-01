create table public."group"
(
    id uuid not null,
    parent_id uuid not null,
    name text not null
        constraint group_unique_name
            unique,
    flags integer default 0 not null,
    key text not null,
    constraint group_pk
        unique (id, parent_id)
);

alter table public."group" owner to lazyfingers;

create unique index group_id_uindex
    on public."group" (id);

create index group_flags_index
    on public."group" (flags);

create unique index group_key_uindex
    on public."group" (key);

create table public.group_assets
(
    group_id uuid not null,
    asset_id uuid not null,
    asset_kind smallint default 0 not null,
    constraint group_assets_pk
        primary key (group_id, asset_id, asset_kind)
);

alter table public.group_assets owner to lazyfingers;

create index group_assets_asset_id_index
    on public.group_assets (asset_id);

create index group_assets_group_id_index
    on public.group_assets (group_id);

create table public.accesspolicy_roster
(
    policy_id uuid not null,
    actor_kind smallint not null,
    actor_id uuid not null,
    access bigint not null,
    access_explained text,
    constraint accesspolicy_roster_pk
        primary key (policy_id, actor_kind, actor_id)
);

alter table public.accesspolicy_roster owner to lazyfingers;

create index accesspolicy_roster_policy_id_actor_kind_index
    on public.accesspolicy_roster (policy_id, actor_kind);

create index accesspolicy_roster_policy_id_index
    on public.accesspolicy_roster (policy_id);

create table public.password
(
    kind smallint not null,
    owner_id uuid not null,
    hash bytea not null,
    is_change_required boolean default false not null,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone,
    expire_at timestamp with time zone,
    constraint password_pk
        primary key (kind, owner_id)
);

alter table public.password owner to lazyfingers;

create table public.user_email
(
    user_id uuid not null,
    addr text not null
        constraint user_email_pk
            primary key,
    is_primary boolean default false not null,
    created_at timestamp not null,
    confirmed_at timestamp,
    updated_at timestamp
);

alter table public.user_email owner to lazyfingers;

create index user_email_user_id_index
    on public.user_email (user_id);

create table public.user_phone
(
    user_id uuid not null,
    number text not null,
    is_primary boolean default false,
    created_at timestamp not null,
    confirmed_at timestamp,
    updated_at timestamp,
    constraint user_phone_pk
        primary key (user_id, number)
);

alter table public.user_phone owner to lazyfingers;

create index user_phone_user_id_index
    on public.user_phone (user_id);

create unique index user_phone_number_uindex
    on public.user_phone (number);

create table public.user_profile
(
    user_id uuid not null
        constraint user_profile_pk
            primary key,
    firstname text,
    middlename text,
    lastname text,
    language text,
    checksum numeric default 0 not null,
    created_at timestamp not null,
    updated_at timestamp
);

alter table public.user_profile owner to lazyfingers;

create table public.token
(
    kind smallint not null,
    hash bytea not null
        constraint token_pk
            primary key,
    checkin_total integer not null,
    checkin_remainder integer not null,
    created_at timestamp with time zone not null,
    expire_at timestamp with time zone
);

alter table public.token owner to lazyfingers;

create index token_created_at_index
    on public.token (created_at);

create index token_expire_at_index
    on public.token (expire_at);

create table public."user"
(
    id uuid not null
        constraint user_pk
            primary key,
    username text not null,
    display_name text not null,
    last_login_at timestamp with time zone,
    last_login_ip inet,
    last_login_failed_at timestamp with time zone,
    last_login_failed_ip inet,
    last_login_attempts smallint not null,
    is_suspended boolean default false,
    suspension_reason text,
    suspension_expires_at timestamp with time zone,
    suspended_by_id uuid,
    checksum numeric,
    confirmed_at timestamp with time zone,
    created_at timestamp with time zone,
    created_by_id uuid,
    updated_at timestamp with time zone,
    updated_by_id uuid,
    deleted_at timestamp with time zone,
    deleted_by_id uuid
);

alter table public."user" owner to lazyfingers;

create unique index user_username_uindex
    on public."user" (username);

create unique index user_display_name_uindex
    on public."user" (display_name);

create table public.client
(
    id uuid not null
        constraint client_pk
            primary key,
    name text,
    flags smallint default 0 not null,
    registered_at timestamp with time zone not null,
    expire_at timestamp with time zone,
    urls text[],
    entropy bytea,
    metadata jsonb
);

alter table public.client owner to lazyfingers;

create index client_expire_at_index
    on public.client (expire_at);

create index client_registered_at_index
    on public.client (registered_at);

create index client_name_index
    on public.client (name);

create unique index client_name_uindex
    on public.client (name);

create table public.device
(
    id uuid not null
        constraint device_pk
            primary key,
    name text,
    imei text,
    meid text,
    serial_number text,
    flags smallint default 0 not null,
    registered_at timestamp with time zone not null,
    expire_at timestamp with time zone
);

alter table public.device owner to lazyfingers;

create index device_expire_at_index
    on public.device (expire_at);

create index device_registered_at_index
    on public.device (registered_at);

create index device_name_index
    on public.device (name);

create index device_imei_index
    on public.device (imei);

create index device_meid_index
    on public.device (meid);

create index device_serial_number_index
    on public.device (serial_number);

create table public.device_assets
(
    device_id integer not null,
    asset_kind smallint not null,
    asset_id uuid not null,
    constraint device_relations_pk
        primary key (device_id, asset_kind, asset_id)
);

alter table public.device_assets owner to lazyfingers;

create index device_relations_asset_kind_asset_id_index
    on public.device_assets (asset_kind, asset_id);

create index device_relations_device_id_asset_kind_index
    on public.device_assets (device_id, asset_kind);

create index device_relations_device_id_index
    on public.device_assets (device_id);

create table public.accesspolicy
(
    id uuid not null
        constraint accesspolicy_id_pk
            primary key,
    parent_id uuid,
    owner_id uuid not null,
    key text not null
        constraint accesspolicy_pk
            unique,
    object_name text,
    object_id uuid,
    flags smallint default 0 not null
);

alter table public.accesspolicy owner to lazyfingers;

create unique index accesspolicy__key_uindex
    on public.accesspolicy (key)
    where (btrim(key) <> ''::text);

create unique index accesspolicy_pk_object_name_id
    on public.accesspolicy (key)
    where (btrim(object_name) <> ''::text);

create table public.auth_session
(
    id uuid not null,
    trace_id uuid not null,
    client_id uuid not null
        constraint auth_session_client_id_fk
            references public.client,
    identity_kind text not null,
    identity_id uuid not null,
    ip text not null,
    flags smallint not null,
    created_at timestamp with time zone not null,
    refreshed_at timestamp with time zone,
    revoked_at timestamp with time zone,
    expire_at timestamp with time zone not null,
    revoke_reason text
);

alter table public.auth_session owner to lazyfingers;

create index auth_session_trace_id_index
    on public.auth_session (trace_id);

create table public.auth_refresh_token
(
    id uuid not null
        constraint auth_refresh_token_pk
            primary key,
    trace_id uuid not null,
    parent_id uuid,
    rotated_id uuid,
    last_session_id uuid not null,
    client_id uuid not null
        constraint auth_refresh_token_client_id_fk
            references public.client,
    identity jsonb not null,
    hash bytea not null,
    created_at timestamp with time zone not null,
    rotated_at timestamp with time zone,
    revoked_at timestamp with time zone,
    expire_at timestamp with time zone,
    flags smallint default 0 not null
);

alter table public.auth_refresh_token owner to lazyfingers;

create unique index auth_refresh_token_hash_uindex
    on public.auth_refresh_token (hash);

create table public.auth_code_exchange
(
    code text not null
        constraint auth_code_exchange_pk
            primary key,
    trace_id uuid not null,
    pkce_challenge text not null,
    pkce_method text not null,
    access_token text not null,
    refresh_token text not null
);

alter table public.auth_code_exchange owner to lazyfingers;

