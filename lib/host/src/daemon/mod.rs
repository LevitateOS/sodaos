//! Private Unix-socket daemon transport, admission and route owners.

pub mod admission;
pub mod backend;
pub(crate) mod broker;
pub mod config;
pub mod response;
pub mod routes;
pub mod server;

mod http;
mod websocket;
