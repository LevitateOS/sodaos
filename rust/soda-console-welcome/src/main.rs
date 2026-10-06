//! Soda OS host operator console welcome. Read-only, operator-only.
//! Optional config path supports inspection; the installed login hook uses
//! the default. Preserves the predecessor's main-table uplink lookup.
//!
//! Rust port of appliance/bin/soda-console-welcome, including its embedded
//! Python config/origin validation. Std-only (`cargo build`).
//! Every banner line, subprocess argv shape, validation rule and exit code
//! matches the shell original; subprocesses are resolved via PATH so the
//! installed command doubles keep working.

mod config;
mod origin;
#[cfg(test)]
mod tests;
mod welcome;

use crate::welcome::run;

fn main() {
    std::process::exit(run());
}
