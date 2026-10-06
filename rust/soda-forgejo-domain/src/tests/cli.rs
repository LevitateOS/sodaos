use crate::cli::parse_args;

#[test]
fn cli_parsing_matches_argparse_verbs() {
    let argv = |words: &[&str]| words.iter().map(|w| w.to_string()).collect::<Vec<_>>();
    assert_eq!(
        parse_args("p", &argv(&["status"])),
        Ok(Some("status".to_string()))
    );
    assert!(parse_args("p", &argv(&["-h"])).expect("help").is_none());
    assert!(parse_args("p", &argv(&["--help"])).expect("help").is_none());
    assert_eq!(
        parse_args("p", &argv(&[])),
        Err("the following arguments are required: verb".to_string())
    );
    assert_eq!(
        parse_args("p", &argv(&["freeze"])),
        Err("argument verb: invalid choice: 'freeze' (choose from stop, inhibit, status, lift, start)".to_string())
    );
    assert_eq!(
        parse_args("p", &argv(&["status", "extra"])),
        Err("unrecognized arguments: extra".to_string())
    );
    assert_eq!(
        parse_args("p", &argv(&["--bogus", "status"])),
        Err("unrecognized arguments: --bogus".to_string())
    );
}
