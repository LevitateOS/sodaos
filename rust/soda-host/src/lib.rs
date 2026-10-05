//! Privileged Soda project host daemon (Rust).
//!
//! Lane H port of `cmd/soda-host` + `internal/host/*` (minus the retained Go
//! client and server-side libraries). Wire JSON, socket paths, CLI surface
//! and exit statuses match the Go implementation byte for byte.

pub mod account;
pub mod dbackend;
pub mod domain;
pub mod gmux_admission;
pub mod gmux_backend;
pub mod gmux_routes;
pub mod gmux_server;
pub mod iclient;
pub mod iconfig;
pub mod json;
pub mod muse;
pub mod muse_serve;
pub mod net;
pub mod nist;
pub mod pfactory;
pub mod pops;
pub mod preparation;
pub mod prepare;
pub mod project;
pub mod sha256;
pub mod ssh;
pub mod tailnet_companion;
pub mod tailnet_domain;
pub mod tailnet_files;
pub mod tailnet_runtime;
pub mod tcodex;
pub mod tcontrol;
pub mod tcontrol_enroll;
pub mod tcontrol_native;
pub mod tcontrol_policy;
pub mod tcontrol_provider;
pub mod tcontrol_wire;
pub mod texec;
pub mod tfactory;
pub mod tmuse;
