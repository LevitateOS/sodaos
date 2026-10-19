//! Shared recorded-RPM inventory admission for appliance and Docker provenance.
use super::Error;

/// validRPMInventory guards the recorded bill-of-materials shape: sorted,
/// unique NAME EPOCH:VERSION-RELEASE.ARCH lines. It validates what the build
/// observed; it never pins what a future build must install.
pub fn valid_rpm_inventory(lines: &[String]) -> Result<(), Error> {
    // slices.IsSorted is non-decreasing; duplicates fail the shape check below.
    if lines.windows(2).any(|w| w[0] > w[1]) {
        return Err(Error("sorted RPM inventory required".to_owned()));
    }
    let mut seen: Vec<&str> = Vec::new();
    for line in lines {
        if !is_rpm_line(line) || seen.contains(&line.as_str()) {
            return Err(Error("invalid recorded RPM inventory".to_owned()));
        }
        seen.push(line);
    }
    Ok(())
}

fn is_rpm_line(line: &str) -> bool {
    // `^[a-zA-Z0-9][a-zA-Z0-9+._-]* [0-9]+:[a-zA-Z0-9+._~^-]+$`
    let (name, evr) = match line.split_once(' ') {
        Some(pair) => pair,
        None => return false,
    };
    if evr.contains(' ') {
        return false;
    }
    let mut name_bytes = name.bytes();
    match name_bytes.next() {
        Some(b) if b.is_ascii_alphanumeric() => {}
        _ => return false,
    }
    if !name_bytes.all(|b| b.is_ascii_alphanumeric() || matches!(b, b'+' | b'.' | b'_' | b'-')) {
        return false;
    }
    let (epoch, version) = match evr.split_once(':') {
        Some(pair) => pair,
        None => return false,
    };
    // Imported RPM signing keys appear in the native Fedora inventory with
    // no architecture. Keep their observed records without permitting this
    // pseudo-package representation for ordinary packages.
    let version = if name == "gpg-pubkey" {
        version.strip_suffix(".(none)").unwrap_or(version)
    } else {
        version
    };
    if epoch.is_empty()
        || !epoch.bytes().all(|b| b.is_ascii_digit())
        || version.is_empty()
        || !version.bytes().all(|b| {
            b.is_ascii_alphanumeric() || matches!(b, b'+' | b'.' | b'_' | b'~' | b'^' | b'-')
        })
    {
        return false;
    }
    true
}
