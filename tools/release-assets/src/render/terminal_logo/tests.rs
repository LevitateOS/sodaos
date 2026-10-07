use super::*;

fn document(paths: &str) -> String {
    format!(
        "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 128 128\">\n  <title id=\"t\">S</title>\n  <desc id=\"d\">D</desc>\n{paths}\n</svg>\n"
    )
}

#[test]
fn svg_path_parser_accepts_valid_numbers_and_expands_moveto_pairs() {
    let paths = polygons("M+.5 .5 10 0H10V10L0 10Z").unwrap();
    assert_eq!(paths.len(), 1);
    assert_eq!(
        paths[0],
        [
            (0.5, 0.5),
            (10.0, 0.0),
            (10.0, 0.0),
            (10.0, 10.0),
            (0.0, 10.0)
        ]
    );

    let rings = polygons("M0 0L10 0L0 10Z M20 20L30 20L20 30Z").unwrap();
    assert_eq!(rings.len(), 2);
}

#[test]
fn path_geometry_requires_absolute_finite_closed_rings() {
    for path in ["", "M0 0L10 0", "M0 0L10 0M0 10", "L0 0L10 0Z"] {
        assert!(polygons(path).is_err(), "accepted {path:?}");
    }
    assert_eq!(
        polygons("M0 0Z").unwrap_err(),
        "Emblem polygon has fewer than 3 points"
    );
    assert_eq!(
        polygons("M0 0L10 0L0 10Q1 2 3 4Z").unwrap_err(),
        "Unsupported emblem command: Q"
    );
    assert_eq!(
        polygons("M0 0l10 0L0 10Z").unwrap_err(),
        "Unsupported emblem command: l"
    );
    assert!(polygons("M0 0L1e9999 0L0 10Z").is_err());
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
fn xml_gate_checks_resolved_namespaces_and_rejects_dtd_text_and_transforms() {
    let paths = "<path fill=\"#df001b\" fill-rule=\"evenodd\" d=\"M0 0H8V8H0Z\"/><path fill=\"#101010\" fill-rule=\"evenodd\" d=\"M0 0H4V4H0Z\"/>";
    let prefixed = format!(
        "<s:svg xmlns:s=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 128 128\"><s:path fill=\"#df001b\" fill-rule=\"evenodd\" d=\"M0 0H8V8H0Z\"/><s:path fill=\"#101010\" fill-rule=\"evenodd\" d=\"M0 0H4V4H0Z\"/></s:svg>"
    );
    assert!(render_svg(&prefixed).is_ok());

    let reset = document(&paths.replace("<path", "<path xmlns=\"\""));
    assert!(render_svg(&reset)
        .unwrap_err()
        .contains("Unexpected emblem element"));
    assert!(
        render_svg(&document(&paths).replace("<svg ", "<svg transform=\"scale(2)\" "))
            .unwrap_err()
            .contains("attribute")
    );
    assert!(
        render_svg(&document(&paths).replace("<path ", "<path transform=\"scale(2)\" "))
            .unwrap_err()
            .contains("attribute")
    );
    assert!(render_svg(&document(&paths).replace("</svg>", "unowned text</svg>")).is_err());
    assert!(render_svg(&format!(
        "<!DOCTYPE svg [<!ENTITY mark \"expanded\">]>{}",
        document(&paths)
    ))
    .is_err());
    assert!(render_svg(&" ".repeat(64 * 1024 + 1)).is_err());
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
