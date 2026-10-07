//! Standard IP parsing and formatting with explicit installer zone, prefix,
//! and address-selection policy.

pub use self::address::{parse_addr, parse_addr_bytes, Addr, NetipError};
pub use self::prefix::{parse_prefix, Prefix};

mod address;
mod address_format;
mod prefix;

#[cfg(test)]
mod tests;
