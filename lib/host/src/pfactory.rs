//! Compatibility shell: the native factory owner is now [`crate::factory`].
//! Retained so existing `pfactory::` paths keep resolving; no caller edits.

pub use crate::factory::*;
