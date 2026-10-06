//! Go `net/netip` subset: `ParseAddr`/`ParsePrefix` with Go's exact
//! accept/reject behavior and error text, `Is4`/`Is6`/`Is4In6`/`IsPrivate`,
//! canonical `String`, `Masked`, prefix `Contains`.
//!
//! Byte fidelity: Go strings are byte sequences, and `ParseAddr` operates on
//! bytes (a zone may hold arbitrary non-UTF-8 bytes after URL unescaping, and
//! parse errors quote the raw input). The parser core therefore runs on
//! `&[u8]`; [`parse_addr`] is a thin `&str` wrapper. Zones are unbounded and
//! unvalidated past non-emptiness, as in Go. `GODEBUG` has no `netip` knobs.

pub use self::address::{parse_addr, parse_addr_bytes, Addr, NetipError};
// Prefix stays reachable as netip::Prefix; no in-crate namer exists yet.
#[allow(unused_imports)]
pub use self::prefix::{parse_prefix, Prefix};

mod address;
mod address_format;
mod prefix;

#[cfg(test)]
mod tests;
