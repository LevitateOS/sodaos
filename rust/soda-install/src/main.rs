// soda-install is an explicit operator console adapter, not a boot-time
// installer daemon. The disk subcommand refuses non-live CoreOS hosts.
mod buildx;
mod candidate;
mod command;
mod console;
mod deliver;
mod disks;
mod enroll;
mod errors;
mod execute;
mod fmtx;
mod hostadmit;
mod inputs;
mod jsongo;
mod netip;
mod oci;
mod pathx;
mod pemx;
mod run;
mod setup;
mod signal;
mod sshkey;
mod urlx;
mod wizard;
mod x509;

fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.len() != 2 || !action(&args[1]) {
        eprintln!("usage: soda-install disk|configure|enroll-key|enrollment-serve|enrollment-receive");
        std::process::exit(2);
    }
    crate::signal::install_handlers();
    let ctx = crate::signal::Ctx::root(args[1] != "disk");
    let runner = crate::command::RealRunner;
    if let Err(err) = crate::run::run(&ctx, &runner, &args[1]) {
        eprintln!("{err}");
        std::process::exit(1);
    }
}

fn action(value: &str) -> bool {
    matches!(
        value,
        "disk" | "configure" | "enroll-key" | "enrollment-serve" | "enrollment-receive"
    )
}
