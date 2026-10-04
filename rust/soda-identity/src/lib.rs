// Soda subscription identity broker: Unix HTTP service over the private
// admin and runtime sockets. Ports cmd/soda-identity plus
// internal/identity/control and the identity store surface; the Go
// identity client, domain records and host callbacks are unchanged.
pub mod control;
pub mod crypto;
pub mod http;
pub mod pg;
pub mod runtime;
pub mod schema;
pub mod store;
pub mod strict;
pub mod wire;
