//! Standard IP parsing and formatting with explicit installer zone, prefix,
//! and address-selection policy.

pub use self::address::{parse_addr, Addr};
pub use self::prefix::parse_prefix;

mod address;
mod address_format;
mod prefix;

#[cfg(test)]
mod tests;
