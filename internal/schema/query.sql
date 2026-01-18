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

-- name: ListEntities :many
select id, canonical_id from entities
order by id;

-- name: GetEntityByCanonicalID :one
select id, canonical_id from entities
where canonical_id = ?;

-- name: ListHeartbeats :many
select h.id, h.entity_id, h.ts, h.successful, h.error, e.canonical_id
from heartbeat h
join entities e on h.entity_id = e.id
where (e.canonical_id = ?1 or ?1 is null)
  and (h.ts >= ?2 or ?2 is null)
  and (h.ts <= ?3 or ?3 is null)
  and (h.successful = ?4 or ?4 is null)
order by h.ts desc
limit ?5 offset ?6;

-- name: ListMetrics :many
select m.id, m.entity_id, m.ts, m.name, m.type, m.value, m.labels, e.canonical_id
from metrics m
join entities e on m.entity_id = e.id
where (e.canonical_id = ?1 or ?1 is null)
  and (m.ts >= ?2 or ?2 is null)
  and (m.ts <= ?3 or ?3 is null)
  and (m.name = ?4 or ?4 is null)
  and (m.type = ?5 or ?5 is null)
order by m.ts desc
limit ?6 offset ?7;
