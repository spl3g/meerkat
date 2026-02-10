-- name: GetEntityID :one
select id from entities
 where canonical_id = ?;

-- name: GetCanonicalID :one
select canonical_id from entities
 where id = ?;

-- name: InsertEntity :one
insert into entities(canonical_id)
values (?)
returning id;

-- name: InsertHeartbeat :one
insert into heartbeat(entity_id, ts, successful, error)
values (?, ?, ?, ?)
returning id;

-- name: InsertMetrics :one
insert into metrics(entity_id, ts, name, type, value, labels)
values (?, ?, ?, ?, ?, ?)
returning id;
