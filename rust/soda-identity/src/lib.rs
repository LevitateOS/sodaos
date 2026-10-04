// Soda subscription identity broker: Unix HTTP service over the private
// admin and runtime sockets. Ports cmd/soda-identity plus
// internal/identity/control and the identity store surface; the Go
// identity client, domain records and host callbacks are unchanged.
pub mod strict;
pub mod wire;
