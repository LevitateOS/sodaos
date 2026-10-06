//! Oracle tests for the PR26 daemon-foundation transport: the identity
//! broker client (`soda_host::iclient`).
//!
//! Request bodies are pinned against bytes captured from the live Go
//! client; responses against the Go transport. Daemon config goldens
//! live with their subject in `src/iconfig/tests.rs`.

use soda_host::iclient::{execution_is_terminal, BrokerClient, Execution};
use soda_host::{muse, terminal};
use std::time::{Duration, Instant};

mod common;
mod requests;
mod responses;
