create table entities(
  id integer primary key,
  canonical_id text not null
);

create index if not exists heartbeat_service_id_index on entities (canonical_id);

create table observations(
  id integer primary key,
  entity_id integer references entities,
  ts timestamp not null,
  kind text not null
);

create table heartbeat(
  obs_id integer primary key references observations,
  successful boolean not null,
  error text
);

