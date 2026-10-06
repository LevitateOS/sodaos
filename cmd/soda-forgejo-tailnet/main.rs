// Thin soda-forgejo-tailnet entry. U1 wires the private implementation so
// its inline tests compile; U2 implements the exit-contract call.
#[allow(dead_code)]
#[path = "../../lib/host/src/tailnet/forgejo.rs"]
mod forgejo;

fn main() {}
