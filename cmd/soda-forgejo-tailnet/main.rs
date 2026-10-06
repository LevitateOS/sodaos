// Thin soda-forgejo-tailnet entry calling the private implementation in
// lib/host/src/tailnet/forgejo.rs. Same installed executable identity and
// exit-1-on-error shape as the retired Go helper; stderr diagnostic bytes
// intentionally differ (subprocess stderr is suppressed, see forgejo.rs),
// so no byte-for-byte stderr parity is claimed.
#[path = "../../lib/host/src/tailnet/forgejo.rs"]
mod forgejo;

fn main() {
    if let Err(e) = forgejo::run(&soda_host::project::Native) {
        eprintln!("{e}");
        std::process::exit(1);
    }
}
