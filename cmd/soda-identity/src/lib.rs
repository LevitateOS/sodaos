// Soda subscription identity broker: Unix HTTP service over the private
// admin and runtime sockets. Ports cmd/soda-identity plus
// internal/identity/control and the identity store surface; the Go
// identity client, domain records and host callbacks are unchanged.
pub mod acquisition;
pub mod control;
pub mod crypto;
pub mod enrollment;
pub mod grants;
pub mod http;
pub mod http_routes;
pub mod http_wire;
pub mod pg;
pub mod pg_dsn;
pub mod pg_query;
pub mod providers;
pub mod registration;
pub mod retirement;
pub mod runtime;
pub mod schema;
pub mod store;
pub mod store_connections;
pub mod store_events;
pub mod store_executions;
pub mod store_grants;
pub mod store_leases;
pub mod store_schema;
pub mod strict;
pub mod wire;
pub mod wire_errors;
pub mod wire_execution;
pub mod wire_grants;
pub mod wire_scalars;
pub mod wire_time;
