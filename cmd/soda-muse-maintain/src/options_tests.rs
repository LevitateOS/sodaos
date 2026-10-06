use crate::test_support::PROJECT;

use super::{parse, usage_text};

fn args(v: &[&str]) -> Vec<String> {
    v.iter().map(|s| s.to_string()).collect()
}

#[test]
fn flag_parsing_vectors() {
    let o = parse(&args(&["--project", PROJECT])).unwrap();
    assert_eq!(o.config, "/etc/soda/host.json");
    assert_eq!(o.project, PROJECT);
    assert_eq!(o.tools, "/usr/share/soda/muse-tools");
    assert!(!o.bind_only);
    let o = parse(&args(&[
        "--config=/x/host.json",
        "--tools",
        "/y/tools",
        "--bind-only",
        "--project",
        PROJECT,
    ]))
    .unwrap();
    assert_eq!(o.config, "/x/host.json");
    assert_eq!(o.tools, "/y/tools");
    assert!(o.bind_only);
    let o = parse(&args(&["-project", PROJECT, "-bind-only=false"])).unwrap();
    assert!(!o.bind_only);
    let o = parse(&args(&["--project", PROJECT, "--", "--bind-only"]));
    assert!(o.is_err());
    // First non-flag argument stops parsing.
    assert!(parse(&args(&["--project", PROJECT, "extra"])).is_err());
    assert!(parse(&args(&["extra", "--project", PROJECT])).is_err());
    // Usage errors print usage and name the flag.
    assert_eq!(
        parse(&args(&["--project"])).unwrap_err(),
        "flag needs an argument: -project"
    );
    assert_eq!(
        parse(&args(&["--nope", "x", "--project", PROJECT])).unwrap_err(),
        "flag provided but not defined: -nope"
    );
    assert_eq!(
        parse(&args(&["---project", PROJECT])).unwrap_err(),
        "bad flag syntax: ---project"
    );
    assert_eq!(parse(&args(&["-=x"])).unwrap_err(), "bad flag syntax: -=x");
    assert_eq!(
        parse(&args(&["--bind-only=maybe"])).unwrap_err(),
        "invalid boolean value \"maybe\" for -bind-only: parse error"
    );
    assert_eq!(parse(&args(&["-h"])).unwrap_err(), "flag: help requested");
    assert_eq!(
        usage_text("/usr/share/soda/muse-tools"),
        "Usage of soda-muse-maintain:\n  -bind-only\n    \trestore only the launch interface\n  -config string\n    \toperator host configuration (default \"/etc/soda/host.json\")\n  -project string\n    \texact project identity\n  -tools string\n    \tinstalled public tool directory (default \"/usr/share/soda/muse-tools\")\n"
    );
    // Validation errors.
    assert_eq!(
        parse(&args(&["--project", "../foreign"])).unwrap_err(),
        "explicit project and absolute maintenance paths required"
    );
    assert_eq!(
        parse(&args(&["--project", "P0123456789ABCDEF01234567"])).unwrap_err(),
        "explicit project and absolute maintenance paths required"
    );
    assert_eq!(
        parse(&args(&["--project", PROJECT, "--config", "relative.json"])).unwrap_err(),
        "explicit project and absolute maintenance paths required"
    );
    assert_eq!(
        parse(&args(&[])).unwrap_err(),
        "explicit project and absolute maintenance paths required"
    );
}
