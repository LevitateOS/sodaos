//! Private browser setup: address selection, the Forgejo operator token
//! flow through the native setup/activation binaries, and read-only
//! configured-access guidance.

use crate::command::Runner;
use crate::console::Console;
use crate::errors::Error;
use crate::signal::Ctx;

use self::configure::configure_private_install;

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
