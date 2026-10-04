// One explicitly requested target-side payload phase; piped over SSH.
fn main() {
    eprintln!("usage: soda-acceptance-remote native-phase|cockpit-account|project-state");
    std::process::exit(1);
}
