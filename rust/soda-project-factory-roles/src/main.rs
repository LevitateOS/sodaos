#[path = "../../../cmd/soda-project-terminal/src/factory_roles/mod.rs"]
mod factory_roles;

pub(crate) use factory_roles::*;

fn main() {
    std::process::exit(factory_roles::run());
}
