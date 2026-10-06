use crate::cli::parse_args;

#[test]
fn cli_parsing_matches_argparse() {
    let argv = |words: &[&str]| words.iter().map(|w| w.to_string()).collect::<Vec<_>>();
    let args = parse_args(
        "soda-activate",
        &argv(&["--bind-ip", "192.168.1.5", "--local-tls"]),
    )
    .expect("ok")
    .expect("args");
    assert!(args.local_tls);
    assert_eq!(args.bind_ip, "192.168.1.5");
    // Unambiguous prefixes and --opt=value work like argparse.
    let args = parse_args("soda-activate", &argv(&["--bind=192.168.1.5", "--local"]))
        .expect("ok")
        .expect("args");
    assert_eq!(args.bind_ip, "192.168.1.5");
    assert!(parse_args("soda-activate", &argv(&["-h"]))
        .expect("help")
        .is_none());
    assert!(parse_args("soda-activate", &argv(&["--help"]))
        .expect("help")
        .is_none());
    assert_eq!(
        parse_args("soda-activate", &argv(&["--local-tls"])),
        Err("the following arguments are required: --bind-ip".to_string())
    );
    assert!(parse_args("soda-activate", &argv(&["--bind-ip", "x", "--bogus"])).is_err());
    assert!(parse_args("soda-activate", &argv(&["--bind-ip"])).is_err());
    assert!(parse_args("soda-activate", &argv(&["--bind-ip", "x", "positional"])).is_err());
    assert!(parse_args("soda-activate", &argv(&["--local-tls=x", "--bind-ip", "y"])).is_err());
}
