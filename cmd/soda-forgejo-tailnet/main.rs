// Thin soda-forgejo-tailnet entry calling the private implementation in
// lib/host/src/tailnet/forgejo.rs. Installed executable identity and the
// stderr/exit-1 failure contract match the Go helper byte for byte.
#[path = "../../lib/host/src/tailnet/forgejo.rs"]
mod forgejo;

fn main() {
    if let Err(e) = forgejo::run(&soda_host::project::Native) {
        eprintln!("{e}");
        std::process::exit(1);
    }
}
