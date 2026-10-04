//! Privileged Soda project host daemon (Rust).
//!
//! Lane H port of `cmd/soda-host` + `internal/host/*` (minus the retained Go
//! client and server-side libraries). Wire JSON, socket paths, CLI surface
//! and exit statuses match the Go implementation byte for byte.

pub mod domain;
pub mod json;
pub mod net;
pub mod project;
pub mod sha256;
pub mod ssh;
