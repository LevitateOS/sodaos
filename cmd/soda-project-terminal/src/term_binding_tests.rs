use super::*;
use crate::term_binding::{
    binding_matches_account, read_reservation, validate_binding, validate_reservation,
};
use crate::term_status::{classify_bare_state, UnitClass};

pub(crate) fn sample() -> Account {
    Account {
        pw_name: "op".to_string(),
        pw_uid: 1001,
        pw_gid: 1002,
        pw_dir: "/home/op".to_string(),
        pw_shell: "/bin/bash".to_string(),
    }
}

pub(crate) fn binding_doc(
    account: &str,
    identity: &str,
    cols: &str,
    rows: &str,
    created: &str,
) -> String {
    format!(
        "{{\"account\":{account},\"identity\":{identity},\"cols\":{cols},\"rows\":{rows},\"created_at\":{created}}}"
    )
}

pub(crate) fn good_account() -> String {
    r#"["op",1001,1002,"/home/op","/bin/bash"]"#.to_string()
}

pub(crate) fn parse(text: &str) -> String {
    text.to_string()
}

#[test]
fn path_matrix() {
    let id = "a".repeat(32);
    assert_eq!(
        terminal_path(&id).unwrap(),
        format!("/run/soda-terminals/{id}")
    );
    assert!(terminal_path("").is_err());
    assert!(terminal_path(&"A".repeat(32)).is_err());
    assert!(terminal_path("short").is_err());
    assert!(terminal_path("../escape").is_err());
}

#[test]
fn binding_matrix() {
    let good = binding_doc(&good_account(), "7", "80", "24", "1700000000");
    let record = validate_binding(&parse(&good)).unwrap();
    assert_eq!(record.login, "op");
    assert_eq!((record.uid, record.gid), (1001, 1002));
    assert_eq!(
        (record.identity, record.cols, record.rows, record.created_at),
        (7, 80, 24, 1700000000)
    );
    let negative_zero_gid = validate_binding(&parse(
        r#"{"account":["op",1001,-0,"/home/op","/bin/bash"],"identity":7,"cols":80,"rows":24,"created_at":1700000000}"#,
    ))
    .unwrap();
    assert_eq!(negative_zero_gid.gid, 0);
    // Shape violations.
    assert!(validate_binding(&parse("[1,2]")).is_err());
    assert!(
        validate_binding(&parse(r#"{"account":[],"identity":1,"cols":80,"rows":24}"#)).is_err()
    );
    assert!(validate_binding(&parse(
        &binding_doc(&good_account(), "7", "80", "24", "1700000000").replace("cols", "colz")
    ))
    .is_err());
    // Duplicate keys tolerated (json.loads last-wins), unlike control frames.
    let dup = binding_doc(&good_account(), "7", "80", "24", "1");
    let dup = dup.replace("\"created_at\":1", "\"created_at\":1,\"created_at\":9");
    assert_eq!(validate_binding(&parse(&dup)).unwrap().created_at, 9);
    let overwritten_bad = binding_doc(&good_account(), "7", "80", "24", "1700000000")
        .replace("\"cols\":80", "\"cols\":1e400,\"cols\":80");
    assert!(validate_binding(&parse(&overwritten_bad)).is_ok());
    let final_bad = binding_doc(&good_account(), "7", "80", "24", "1700000000")
        .replace("\"cols\":80", "\"cols\":80,\"cols\":1e400");
    assert!(validate_binding(&parse(&final_bad)).is_err());
    // Account vector violations.
    for bad in [
        r#"["Root",1001,1002,"/home/op","/bin/bash"]"#, // login case
        r#"["root",1001,1002,"/home/op","/bin/bash"]"#, // root
        r#"["op",0,1002,"/home/op","/bin/bash"]"#,      // uid 0
        r#"["op",-1,1002,"/home/op","/bin/bash"]"#,     // uid negative
        r#"["op",1001,-1,"/home/op","/bin/bash"]"#,     // gid negative
        r#"["op",1001,1002,"home/op","/bin/bash"]"#,    // relative dir
        r#"["op",1001,1002,"/home/op","bin/bash"]"#,    // relative shell
        r#"["op",1001,1002,"/home/op"]"#,               // short
        r#"["op",1001.0,1002,"/home/op","/bin/bash"]"#, // float uid
        r#"["op",true,1002,"/home/op","/bin/bash"]"#,   // bool uid
        r#""op""#,                                      // not a list
    ] {
        assert!(
            validate_binding(&parse(&binding_doc(bad, "7", "80", "24", "1700000000"))).is_err(),
            "{bad}"
        );
    }
    // Scalar violations.
    for (identity, cols, rows, created) in [
        ("0", "80", "24", "1700000000"),       // identity 0
        ("-3", "80", "24", "1700000000"),      // identity negative
        ("7.0", "80", "24", "1700000000"),     // identity float
        ("7", "1", "24", "1700000000"),        // cols small
        ("7", "501", "24", "1700000000"),      // cols big
        ("7", "80", "301", "1700000000"),      // rows big
        ("7", "\"80\"", "24", "1700000000"),   // cols string
        ("7", "80", "24", "0"),                // created 0
        ("7", "80", "24", "-0"),               // created negative zero
        ("7e0", "80", "24", "1700000000"),     // identity exponent
        ("7", "80", "24", "-1"),               // created negative
        ("7", "80", "24", "9007199254740992"), // created > 2^53-1
        ("7", "80", "24", "true"),             // created bool
    ] {
        assert!(
            validate_binding(&parse(&binding_doc(
                &good_account(),
                identity,
                cols,
                rows,
                created
            )))
            .is_err(),
            "{identity}/{cols}/{rows}/{created}"
        );
    }
    // Upper bound 2^53-1 accepted.
    assert!(validate_binding(&parse(&binding_doc(
        &good_account(),
        "7",
        "80",
        "24",
        "9007199254740991"
    )))
    .is_ok());
}

#[test]
fn account_match_matrix() {
    let record = validate_binding(&parse(&binding_doc(
        &good_account(),
        "7",
        "80",
        "24",
        "1700000000",
    )))
    .unwrap();
    let account = sample();
    assert!(binding_matches_account(&record, &account));
    let mut other = account.clone();
    other.pw_uid = 1003;
    assert!(!binding_matches_account(&record, &other));
    let mut other = account.clone();
    other.pw_shell = "/bin/sh".to_string();
    assert!(!binding_matches_account(&record, &other));
}

#[test]
fn reservation_matrix() {
    let scope = "c".repeat(64);
    let good = format!("{{\"expires\":9999999999,\"scope\":\"{scope}\"}}");
    let permit = validate_reservation(&parse(&good)).unwrap();
    assert_eq!((permit.expires, permit.scope), (9999999999, scope));
    for bad in [
        r#"{"expires":9999999999}"#.to_string(),
        r#"{"expires":0,"scope":""}"#.to_string(),
        format!("{{\"expires\":-0,\"scope\":\"{}\"}}", "c".repeat(64)),
        format!("{{\"expires\":-1,\"scope\":\"{}\"}}", "c".repeat(64)),
        format!("{{\"expires\":99.5,\"scope\":\"{}\"}}", "c".repeat(64)),
        format!("{{\"expires\":9e1,\"scope\":\"{}\"}}", "c".repeat(64)),
        r#"{"expires":99,"scope":"short"}"#.to_string(),
        format!("{{\"expires\":99,\"scope\":\"{}\"}}", "C".repeat(64)),
        format!(
            "{{\"expires\":99,\"scope\":\"{}\",\"x\":1}}",
            "c".repeat(64)
        ),
        r#"[]"#.to_string(),
    ] {
        assert!(validate_reservation(&parse(&bad)).is_err(), "{bad}");
    }
}

#[test]
fn missing_reservation_is_none() {
    // Absence probe works without root-owned files.
    let dir = std::env::temp_dir().join(format!("soda-pt-term-{}-res", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let fd = std::fs::File::open(&dir).unwrap();
    assert_eq!(read_reservation(&fd).unwrap(), None);
    assert!(!record_exists(&fd, "reservation").unwrap());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn classify_matrix() {
    use UnitClass::*;
    // Stopped units: live permit → opening, else ended.
    assert_eq!(
        classify_bare_state("inactive", false, false).unwrap(),
        Ended
    );
    assert_eq!(classify_bare_state("failed", true, false).unwrap(), Ended);
    assert_eq!(
        classify_bare_state("inactive", true, true).unwrap(),
        Opening
    );
    // Live units with a permit are corrupt.
    for state in [
        "active",
        "activating",
        "deactivating",
        "reloading",
        "unknown",
    ] {
        assert!(classify_bare_state(state, true, true).is_err(), "{state}");
        assert!(classify_bare_state(state, true, false).is_err(), "{state}");
    }
    // Live states without a permit.
    assert_eq!(
        classify_bare_state("active", false, false).unwrap(),
        ReadyCheck
    );
    assert_eq!(
        classify_bare_state("activating", false, false).unwrap(),
        Opening
    );
    assert_eq!(
        classify_bare_state("deactivating", false, false).unwrap(),
        Ending
    );
    assert!(classify_bare_state("reloading", false, false).is_err());
    assert!(classify_bare_state("", false, false).is_err());
}
