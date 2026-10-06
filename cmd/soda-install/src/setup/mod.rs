//! Private browser setup: address selection, the Forgejo operator token
//! flow through the native setup/activation binaries, and read-only
//! configured-access guidance.

use crate::command::Runner;
use crate::console::Console;
use crate::errors::Error;
use crate::signal::Ctx;

use self::access::configured_access;
use self::address::setup_addresses;
pub use self::address::{private_setup_origin, SetupAddress};
use self::configure::{
    configure_private_install, execute_setup_and_activation, valid_operator_token,
};
use self::local_ca::local_ca_fingerprint;

const LOCAL_CA_PATH: &str = "/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt";
/// `platform.Sbin`: native Soda binaries live here.
const SBIN: &str = "/usr/bin";

// configureInstall uses the existing native Forgejo installer and setup command.
// Soda does not create a second account/password authority or expose the unfinished
// Forgejo installer. It runs in any interactive operator terminal, local or SSH,
// so the token can be typed or pasted; no SSH session is required.
pub fn configure_install(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<(), Error> {
    configure_private_install(ctx, console, run, "/etc/soda", "/run", LOCAL_CA_PATH)
}

mod access;
mod address;
mod configure;
mod local_ca;

#[cfg(test)]
mod tests;
