use super::*;

fn document(paths: &str) -> String {
    format!(
        "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 128 128\">\n  <title id=\"t\">S</title>\n  <desc id=\"d\">D</desc>\n{paths}\n</svg>\n"
    )
}

#[test]
fn tokenizer_matches_findall_shapes() {
    let tokens = tokenize("M40 0H128V88L-8.5.5+3Z");
    let texts: Vec<String> = tokens
        .iter()
        .map(|token| match token {
            Token::Cmd(c) => c.to_string(),
            Token::Num(raw) => raw.clone(),
        })
        .collect();
    // Like findall: the lone `.` and `+` are skipped, not signed fractions.
    assert_eq!(
        texts,
        ["M", "40", "0", "H", "128", "V", "88", "L", "-8.5", "5", "3", "Z"]
            .iter()
            .map(|s| s.to_string())
            .collect::<Vec<String>>()
    );
}

#[test]
fn polygon_gates_match_the_script() {
    assert!(polygons("M0 0L10 0L0 10Z").is_ok());
    assert_eq!(
        polygons("M0 0L10 0").unwrap_err(),
        "Unterminated emblem geometry"
    );
    assert_eq!(polygons("").unwrap_err(), "Unterminated emblem geometry");
    assert_eq!(
        polygons("M0 0L10 0L0 10Z M0 0").unwrap_err(),
        "Unterminated emblem geometry"
    );
    assert_eq!(
        polygons("M0 0Z").unwrap_err(),
        "Emblem polygon has fewer than 3 points"
    );
    assert_eq!(
        polygons("M0 0L10").unwrap_err(),
        "Truncated emblem command: L"
    );
    assert_eq!(polygons("H").unwrap_err(), "Truncated emblem command: H");
    assert_eq!(
        polygons("V5X").unwrap_err(),
        "Unsupported emblem command: X"
    );
    assert_eq!(
        polygons("M0 0L10 0L0 10Z Q1 2").unwrap_err(),
        "Unsupported emblem command: Q"
    );
    assert_eq!(polygons("5").unwrap_err(), "Unsupported emblem command: 5");
}

#[test]
fn svg_gate_rejects_changed_syntax() {
    let good = "  <path fill=\"#df001b\" fill-rule=\"evenodd\" d=\"M0 0H8V8H0Z\"/>\n  <path fill=\"#101010\" fill-rule=\"evenodd\" d=\"M0 0H4V4H0Z\"/>";
    assert!(render_svg(&document(good)).is_ok());
    let swapped = "  <path fill=\"#101010\" fill-rule=\"evenodd\" d=\"M0 0H8V8H0Z\"/>\n  <path fill=\"#df001b\" fill-rule=\"evenodd\" d=\"M0 0H4V4H0Z\"/>";
    assert_eq!(
        render_svg(&document(swapped)).unwrap_err(),
        "Unexpected emblem layers"
    );
    let rule = "  <path fill=\"#df001b\" fill-rule=\"nonzero\" d=\"M0 0H8V8H0Z\"/>\n  <path fill=\"#101010\" fill-rule=\"evenodd\" d=\"M0 0H4V4H0Z\"/>";
    assert_eq!(
        render_svg(&document(rule)).unwrap_err(),
        "Unexpected emblem fill rule"
    );
    assert_eq!(
        render_svg(&document(good).replace("0 0 128 128", "0 0 64 64")).unwrap_err(),
        "Unexpected emblem viewBox"
    );
    assert_eq!(
        render_svg(&document(good).replace("http://www.w3.org/2000/svg", "http://example.invalid"))
            .unwrap_err(),
        "Unexpected emblem layers"
    );
    let grouped = format!(
        "{good}\n  <g><path fill=\"#df001b\" fill-rule=\"evenodd\" d=\"M0 0H8V8H0Z\"/></g>"
    );
    assert!(render_svg(&document(&grouped))
        .unwrap_err()
        .contains("Unexpected emblem element"));
}

#[test]
fn tiny_emblem_renders_both_layers() {
    let paths = "  <path fill=\"#df001b\" fill-rule=\"evenodd\" d=\"M0 0H128V128H0Z\"/>\n  <path fill=\"#101010\" fill-rule=\"evenodd\" d=\"M0 0H64V128H0Z\"/>";
    let (colored, plain) = render_svg(&document(paths)).expect("render");
    let rows: Vec<&str> = plain.lines().collect();
    assert_eq!(rows.len(), 18);
    assert_eq!(rows[0], "@@@@@@@@@@@@@@@@################");
    assert_eq!(rows[15], "@@@@@@@@@@@@@@@@################");
    assert_eq!(rows[16], "");
    assert_eq!(rows[17], "SODA OS");
    assert!(colored
        .lines()
        .next()
        .unwrap()
        .starts_with("$2@@@@@@@@@@@@@@@@$1"));
    assert!(colored.ends_with('\n'));
}
