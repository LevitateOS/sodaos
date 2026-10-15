// ---------- native platform ----------

pub(super) fn oci_architecture(arch: &str) -> Result<&'static str, String> {
    if arch == "x86_64" {
        Ok("amd64")
    } else {
        Err("expected x86_64".to_string())
    }
}

pub(super) fn require_native(arch: &str) -> Result<(), String> {
    let want = oci_architecture(arch)?;
    // `std::env::consts::ARCH` uses rustc names; map to the Go values the
    // payload architecture implies.
    let native = match std::env::consts::ARCH {
        "x86_64" => "amd64",
        "aarch64" => "arm64",
        other => other,
    };
    if std::env::consts::OS != "linux" || native != want {
        return Err("matching-native Linux required".to_string());
    }
    Ok(())
}
