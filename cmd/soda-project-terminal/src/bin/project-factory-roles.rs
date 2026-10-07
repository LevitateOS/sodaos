//! `project-factory-roles`: thin entrypoint for the fixed bounded factory
//! role helper (consolidated from `rust/soda-project-factory-roles`).
//! Compiled installation remains `/usr/libexec/soda/project-factory-roles`.

// The shared modules expose different subsets to each fixed binary entrypoint.
#[allow(dead_code)]
#[path = "../pyemit.rs"]
pub(crate) mod pyemit;
#[allow(dead_code)]
#[path = "../state_json.rs"]
pub(crate) mod state_json;

#[path = "../factory_roles/mod.rs"]
mod factory_roles;

pub(crate) use factory_roles::*;

fn main() {
    std::process::exit(factory_roles::run());
}
