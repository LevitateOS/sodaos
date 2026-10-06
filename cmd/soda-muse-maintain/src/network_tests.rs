use super::parse_prefix;

#[test]
fn prefix_parity_vectors() {
    let cases: &[(&str, Option<&str>)] = &[
        ("10.0.0.0/24", None),
        ("10.0.0.0/0", None),
        ("10.0.0.0/32", None),
        ("::/0", None),
        ("::/128", None),
        ("fe80::1/128", None),
        ("::ffff:1.2.3.4/128", None),
        ("1:2:3:4:5:6:1.2.3.4/128", None),
        (
            "1.2.3.4/33",
            Some("netip.ParsePrefix(\"1.2.3.4/33\"): prefix length out of range"),
        ),
        (
            "::gggg/1",
            Some(
                "netip.ParsePrefix(\"::gggg/1\"): ParseAddr(\"::gggg\"): each colon-separated field must have at least one digit (at \"gggg\")",
            ),
        ),
        ("1.2.3.4", Some("netip.ParsePrefix(\"1.2.3.4\"): no '/'")),
        (
            "1.2.3/24",
            Some("netip.ParsePrefix(\"1.2.3/24\"): ParseAddr(\"1.2.3\"): IPv4 address too short"),
        ),
        (
            "1.2.3.4.5/24",
            Some(
                "netip.ParsePrefix(\"1.2.3.4.5/24\"): ParseAddr(\"1.2.3.4.5\"): IPv4 address too long",
            ),
        ),
        (
            "fe80::1%eth0/64",
            Some(
                "netip.ParsePrefix(\"fe80::1%eth0/64\"): IPv6 zones cannot be present in a prefix",
            ),
        ),
        (
            "1.2.3.4/ab",
            Some("netip.ParsePrefix(\"1.2.3.4/ab\"): bad bits after slash: \"ab\""),
        ),
        (
            "1.2.3.4/",
            Some("netip.ParsePrefix(\"1.2.3.4/\"): bad bits after slash: \"\""),
        ),
        (
            "/24",
            Some("netip.ParsePrefix(\"/24\"): ParseAddr(\"\"): unable to parse IP"),
        ),
        (
            "1.2.3.4/-1",
            Some("netip.ParsePrefix(\"1.2.3.4/-1\"): bad bits after slash: \"-1\""),
        ),
        (
            "::/129",
            Some("netip.ParsePrefix(\"::/129\"): prefix length out of range"),
        ),
        (
            "1::2::3/64",
            Some(
                "netip.ParsePrefix(\"1::2::3/64\"): ParseAddr(\"1::2::3\"): multiple :: in address (at \":3\")",
            ),
        ),
        (
            "1.2.3.256/24",
            Some(
                "netip.ParsePrefix(\"1.2.3.256/24\"): ParseAddr(\"1.2.3.256\"): IPv4 field has value >255",
            ),
        ),
        (
            "12345::/33",
            Some(
                "netip.ParsePrefix(\"12345::/33\"): ParseAddr(\"12345::\"): each group must have 4 or less digits (at \"12345::\")",
            ),
        ),
        (
            "1.2.3.4/ 24",
            Some("netip.ParsePrefix(\"1.2.3.4/ 24\"): bad bits after slash: \" 24\""),
        ),
        (
            "1.2.3.4\x01/24",
            Some(
                "netip.ParsePrefix(\"1.2.3.4\\x01/24\"): ParseAddr(\"1.2.3.4\\x01\"): unexpected character (at \"\\x01\")",
            ),
        ),
        (
            "1.2.3.4/\x034",
            Some("netip.ParsePrefix(\"1.2.3.4/\\x034\"): bad bits after slash: \"\\x034\""),
        ),
        (
            "01.2.3.4/24",
            Some(
                "netip.ParsePrefix(\"01.2.3.4/24\"): ParseAddr(\"01.2.3.4\"): IPv4 field has octet with leading zero",
            ),
        ),
        // Left-to-right order: the value error fires before the length check.
        (
            "1.2.3.99999.5/24",
            Some(
                "netip.ParsePrefix(\"1.2.3.99999.5/24\"): ParseAddr(\"1.2.3.99999.5\"): IPv4 field has value >255",
            ),
        ),
        (
            "1..2.3/24",
            Some(
                "netip.ParsePrefix(\"1..2.3/24\"): ParseAddr(\"1..2.3\"): IPv4 field must have at least one digit (at \".2.3\")",
            ),
        ),
        (
            "1.2.3./24",
            Some(
                "netip.ParsePrefix(\"1.2.3./24\"): ParseAddr(\"1.2.3.\"): IPv4 field must have at least one digit (at \".\")",
            ),
        ),
        (
            "12g4::/64",
            Some(
                "netip.ParsePrefix(\"12g4::/64\"): ParseAddr(\"12g4::\"): unexpected character, want colon (at \"g4::\")",
            ),
        ),
        (
            "1.2.3.4::/64",
            Some(
                "netip.ParsePrefix(\"1.2.3.4::/64\"): ParseAddr(\"1.2.3.4::\"): unexpected character (at \"::\")",
            ),
        ),
        (
            "1:2.3.4.5/24",
            Some(
                "netip.ParsePrefix(\"1:2.3.4.5/24\"): ParseAddr(\"1:2.3.4.5\"): embedded IPv4 address must replace the final 2 fields of the address (at \"2.3.4.5\")",
            ),
        ),
        (
            "::1:2:3:4:5:6:7:1.2.3.4/64",
            Some(
                "netip.ParsePrefix(\"::1:2:3:4:5:6:7:1.2.3.4/64\"): ParseAddr(\"::1:2:3:4:5:6:7:1.2.3.4\"): too many hex fields to fit an embedded IPv4 at the end of the address (at \"1.2.3.4\")",
            ),
        ),
        (
            "::ffff:1.2.3.999/64",
            Some(
                "netip.ParsePrefix(\"::ffff:1.2.3.999/64\"): ParseAddr(\"::ffff:1.2.3.999\"): IPv4 field has value >255",
            ),
        ),
        (
            "fe80::1%/64",
            Some(
                "netip.ParsePrefix(\"fe80::1%/64\"): ParseAddr(\"fe80::1%\"): zone must be a non-empty string",
            ),
        ),
        (
            "abcd%eth0/64",
            Some(
                "netip.ParsePrefix(\"abcd%eth0/64\"): ParseAddr(\"abcd%eth0\"): missing IPv6 address",
            ),
        ),
        (
            "1.2.3.4%eth0/24",
            Some(
                "netip.ParsePrefix(\"1.2.3.4%eth0/24\"): ParseAddr(\"1.2.3.4%eth0\"): unexpected character (at \"%eth0\")",
            ),
        ),
        (
            "1:2:3:4:5:6:7:8:9/64",
            Some(
                "netip.ParsePrefix(\"1:2:3:4:5:6:7:8:9/64\"): ParseAddr(\"1:2:3:4:5:6:7:8:9\"): trailing garbage after address (at \"9\")",
            ),
        ),
        (
            "1:2:3/64",
            Some(
                "netip.ParsePrefix(\"1:2:3/64\"): ParseAddr(\"1:2:3\"): address string too short",
            ),
        ),
        (
            "::1:2:3:4:5:6:7:8/64",
            Some(
                "netip.ParsePrefix(\"::1:2:3:4:5:6:7:8/64\"): ParseAddr(\"::1:2:3:4:5:6:7:8\"): the :: must expand to at least one field of zeros",
            ),
        ),
        (
            "1.2.3.4/00",
            Some("netip.ParsePrefix(\"1.2.3.4/00\"): bad bits after slash: \"00\""),
        ),
        (
            "1.2.3.4/+5",
            Some("netip.ParsePrefix(\"1.2.3.4/+5\"): bad bits after slash: \"+5\""),
        ),
        (
            "1.2.3.4/99999999999999999999999",
            Some(
                "netip.ParsePrefix(\"1.2.3.4/99999999999999999999999\"): bad bits after slash: \"99999999999999999999999\"",
            ),
        ),
    ];
    for (input, expected) in cases {
        match (parse_prefix(input), expected) {
            (Ok(()), None) => {}
            (Err(got), Some(want)) => assert_eq!(&got, want, "input {input:?}"),
            (Ok(()), Some(want)) => panic!("input {input:?}: accepted, want {want}"),
            (Err(got), None) => panic!("input {input:?}: rejected: {got}"),
        }
    }
}
