// soda-setup records the native operator identity and Soda service configuration.

mod cli;
mod config;
mod factory_enrollment;
mod forgejo;
mod json;
mod origin;
mod secrets;
mod setup;
mod system;

use crate::cli::run;

fn main() {
    std::process::exit(run());
}
