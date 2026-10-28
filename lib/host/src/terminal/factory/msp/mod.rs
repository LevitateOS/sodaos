//! The durable native Muse protocol over one already-owned process boundary.
//!
//! This owner never spawns a process, acquires credentials, schedules tasks or
//! appends trusted Factory task events. Its caller supplies the existing admitted
//! run's pipes and retains responsibility for unit/custody retirement. Native
//! child observations remain observations until the admitted-task correlation
//! round trip has been proved; assistant prose cannot create task authority.

mod session;
mod transport;
mod wire;

pub use session::{
    mcp_probe_outcome, new_command_id, sanitize_mcp_servers, SessionSnapshot, TurnAdmission,
};
pub use transport::{Connection, Error, MAX_FRAME_BYTES};
pub use wire::{Item, Notice, Provenance, TurnFailure};
