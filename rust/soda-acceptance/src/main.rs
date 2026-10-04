// Outside support only: invokes existing owned checks; no product scenarios.
use soda_acceptance::driver::{install_signal_forwarding, run};
use soda_acceptance::process::Phase;

fn main() {
    let root = Phase::background();
    install_signal_forwarding(&root);
    let args: Vec<String> = std::env::args().skip(1).collect();
    if let Err(err) = run(&root, &args) {
        eprintln!("{err}");
        std::process::exit(1);
    }
}
