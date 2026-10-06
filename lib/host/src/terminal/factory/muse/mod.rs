//! Muse Code CLI factory runner (`muse exec`).
//!
//! Family adapter to the shared factory core for the `muse` harness family: own
//! admission policy (provider/scope checks, path struct), guest (the
//! staged static `muse` binary — never the auto-updating shell
//! launcher), credential (the broker's opaque `auth.json` bytes staged
//! verbatim at the CLI's file-backend lookup path, exactly like the
//! interactive muse runtime and the enrollment fixture; never parsed,
//! never exported), and supervisor argv. The supervised lifecycle
//! (wait/validate/stop/output/capture/live) is the shared factory
//! core, parameterized by scope and paths.
//! Muse borrows: the broker forgets the lease on return and never calls
//! finish, so the daemon denies `finish` for muse leases (capture only
//! echoes the staged copy for the in-process pfactory return path, whose
//! bytes the broker ignores).

pub mod commands;
pub mod lifecycle;
pub mod paths;
pub mod reserve;
pub mod start;

#[cfg(test)]
mod tests;
