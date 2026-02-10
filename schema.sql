create table if not exists entities(
  id integer primary key,
  canonical_id text not null
);

create index if not exists entities_canonical_id_index on entities (canonical_id);

create table if not exists heartbeat(
  id integer primary key,
  entity_id integer references entities not null,
  ts timestamp not null,
  successful boolean not null,
  error text
);

create table if not exists metrics(
  id integer primary key,
  entity_id integer references entities not null,
  ts timestamp not null,
  name text not null,
  type text not null,
  value real not null,
  labels jsonb not null
);
