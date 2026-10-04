// Host-side helpers answering the kept installed shell skeletons.
use std::io::Read;

use soda_acceptance::host_probes;

fn read_stdin() -> String {
    let mut input = String::new();
    let _ = std::io::stdin().read_to_string(&mut input);
    input
}

fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let outcome = match args
        .iter()
        .map(|arg| arg.as_str())
        .collect::<Vec<_>>()
        .as_slice()
    {
        ["host-content", phase] => host_probes::host_content(phase),
        ["host-listeners", phase] => host_probes::host_listeners(phase),
        ["host-deployments"] => host_probes::host_deployments(&read_stdin()),
        ["operator-tailscale"] => host_probes::operator_tailscale(&read_stdin()),
        ["forgejo-origins"] => host_probes::forgejo_origins(),
        ["forgejo-tailnet"] => host_probes::forgejo_tailnet(&read_stdin()),
        ["forgejo-advertisement"] => host_probes::forgejo_advertisement(),
        _ => {
            eprintln!("usage: soda-host-probes <check> [args]");
            std::process::exit(1);
        }
    };
    match outcome {
        Ok(text) => print!("{text}"),
        Err(failure) => {
            eprintln!("{failure}");
            std::process::exit(1);
        }
    }
}
