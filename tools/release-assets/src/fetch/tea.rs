//! Upstream Tea binary + license staging (`scripts/fetch-tea.py`).

use std::path::{Path, PathBuf};
use std::time::Duration;

pub const RELEASES_API: &str = "https://gitea.com/api/v1/repos/gitea/tea/releases/latest";
pub const DL_BASE: &str = "https://dl.gitea.com/tea";
pub const LICENSE_BASE: &str = "https://gitea.com/gitea/tea/raw/tag";
pub const USER_AGENT: &str = "SodaOS-build";
const TIMEOUT: Duration = Duration::from_secs(60);
const BINARY_LIMIT: u64 = 64_000_000;
const META_LIMIT: u64 = 1_000_000;

/// Overridable endpoints so tests can point the fetcher at a fixture
/// server; production uses [`Endpoints::production`].
pub struct Endpoints {
    pub releases_api: String,
    pub dl_base: String,
    pub license_base: String,
}

impl Endpoints {
    pub fn production() -> Endpoints {
        Endpoints {
            releases_api: RELEASES_API.to_string(),
            dl_base: DL_BASE.to_string(),
            license_base: LICENSE_BASE.to_string(),
        }
    }
}

/// The script's default output rooted at the caller's working directory:
/// the release build always runs fetchers from the source root, so this
/// stages exactly where `ROOT/.artifacts/...` did.
pub fn default_out(arch: &str) -> Result<PathBuf, String> {
    let cwd = std::env::current_dir().map_err(|e| e.to_string())?;
    Ok(cwd
        .join(".artifacts/native")
        .join(arch)
        .join("project-tools"))
}

fn download(url: &str, limit: u64) -> Result<Vec<u8>, String> {
    let mut response = crate::fetch::http_get(url, USER_AGENT, TIMEOUT)
        .map_err(|e| format!("fetch {url}: {e}"))?;
    if !(200..300).contains(&response.status) {
        return Err(format!(
            "fetch {url}: unexpected HTTP status {}",
            response.status
        ));
    }
    crate::fetch::read_capped(&mut response.reader, limit)
}

/// `^v[0-9]+\.[0-9]+\.[0-9]+$` without a regex dependency.
fn valid_tag(tag: &str) -> bool {
    let body = match tag.strip_prefix('v') {
        Some(body) => body,
        None => return false,
    };
    let mut parts = body.split('.');
    match (parts.next(), parts.next(), parts.next(), parts.next()) {
        (Some(a), Some(b), Some(c), None) => {
            !a.is_empty()
                && !b.is_empty()
                && !c.is_empty()
                && a.bytes().all(|b| b.is_ascii_digit())
                && b.bytes().all(|b| b.is_ascii_digit())
                && c.bytes().all(|b| b.is_ascii_digit())
        }
        _ => false,
    }
}

fn latest_tag(endpoints: &Endpoints) -> Result<String, String> {
    let url = &endpoints.releases_api;
    let body = download(url, META_LIMIT)?;
    let text = std::str::from_utf8(&body).map_err(|e| format!("fetch {url}: {e}"))?;
    let release = soda_json::JsonValue::parse(text)
        .map_err(|_| format!("fetch {url}: invalid release metadata"))?;
    let tag = release
        .get("tag_name")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    if !valid_tag(tag) {
        return Err("Tea latest release is not a version tag".to_string());
    }
    Ok(tag.to_string())
}

/// The expected digest for `filename`; the last matching line wins, like
/// the owner's loop without a `break`.
fn checksums_for(text: &str, filename: &str) -> Option<String> {
    let mut expected = None;
    for line in text.split('\n') {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() == 2 && parts[1] == filename {
            expected = Some(parts[0].to_string());
        }
    }
    expected
}

fn valid_elf64(body: &[u8], machine: u16) -> bool {
    if body.len() < 64 || body[..6] != [0x7f, b'E', b'L', b'F', 2, 1] {
        return false;
    }
    u16::from_le_bytes([body[18], body[19]]) == machine
}

/// Stage Tea for `arch` into `out`; returns the stdout line.
pub fn fetch(arch: &str, out: &Path, endpoints: &Endpoints) -> Result<String, String> {
    if arch != "x86_64" {
        return Err("Tea staging supports x86_64 only".to_string());
    }
    let binary = out.join("bin/tea");
    let license_file = out.join("licenses/tea/LICENSE");
    if std::fs::symlink_metadata(&binary).is_ok()
        || std::fs::symlink_metadata(&license_file).is_ok()
    {
        return Err("Tea output already exists; select a fresh output directory".to_string());
    }
    let tag = latest_tag(endpoints)?;
    let version = &tag[1..];
    let filename = format!("tea-{version}-linux-amd64");
    let base = format!("{}/{version}", endpoints.dl_base);
    let sums_url = format!("{base}/checksums.txt");
    let sums = download(&sums_url, META_LIMIT)?;
    let sums_text = std::str::from_utf8(&sums).map_err(|e| format!("fetch {sums_url}: {e}"))?;
    let expected = checksums_for(sums_text, &filename)
        .ok_or_else(|| "Tea checksums omit the requested archive".to_string())?;
    let body = download(&format!("{base}/{filename}"), BINARY_LIMIT)?;
    if body.len() as u64 > BINARY_LIMIT || crate::fetch::sha256_hex(&body) != expected {
        return Err("Tea binary checksum mismatch".to_string());
    }
    if !valid_elf64(&body, 62) {
        return Err("Tea binary is not ELF64 for the requested architecture".to_string());
    }
    let license = download(
        &format!("{}/{tag}/LICENSE", endpoints.license_base),
        META_LIMIT,
    )?;
    if license.is_empty() {
        return Err("Tea license download is empty".to_string());
    }
    std::fs::create_dir_all(out).map_err(|e| e.to_string())?;
    match std::fs::create_dir(out.join("bin")) {
        Ok(()) => {}
        Err(e) if e.kind() == std::io::ErrorKind::AlreadyExists => {}
        Err(e) => return Err(e.to_string()),
    }
    std::fs::write(&binary, &body).map_err(|e| e.to_string())?;
    crate::fetch::chmod(&binary, 0o755)?;
    let license_dir = out.join("licenses/tea");
    std::fs::create_dir_all(&license_dir).map_err(|e| e.to_string())?;
    std::fs::write(&license_file, &license).map_err(|e| e.to_string())?;
    crate::fetch::chmod(&license_file, 0o644)?;
    Ok(format!(
        "Upstream Tea {version} ({arch}) staged at {}",
        out.display()
    ))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::fetch::test_server::{Server, TempDir};
    use std::collections::HashMap;

    const VERSION: &str = "0.99.1";
    const TAG: &str = "v0.99.1";

    fn elf_body(machine: u16) -> Vec<u8> {
        let mut body = vec![0u8; 64];
        body[..6].copy_from_slice(&[0x7f, b'E', b'L', b'F', 2, 1]);
        body[18..20].copy_from_slice(&machine.to_le_bytes());
        body
    }

    struct Fixture {
        server: Server,
        endpoints: Endpoints,
        binary: Vec<u8>,
        license: Vec<u8>,
    }

    impl Fixture {
        fn start() -> Fixture {
            let mut routes: HashMap<String, (u16, Vec<u8>)> = HashMap::new();
            let binary = elf_body(62);
            let license = b"fixture license\n".to_vec();
            let filename = format!("tea-{VERSION}-linux-amd64");
            let sums = format!("{}  {filename}\n", crate::fetch::sha256_hex(&binary));
            routes.insert(
                "/api/releases/latest".to_string(),
                (200, format!(r#"{{"tag_name":"{TAG}"}}"#).into_bytes()),
            );
            routes.insert(
                format!("/{VERSION}/checksums.txt"),
                (200, sums.into_bytes()),
            );
            routes.insert(format!("/{VERSION}/{filename}"), (200, binary.clone()));
            routes.insert(format!("/raw/{TAG}/LICENSE"), (200, license.clone()));
            let server = Server::start(routes);
            let endpoints = Endpoints {
                releases_api: format!("{}/api/releases/latest", server.base),
                dl_base: server.base.clone(),
                license_base: format!("{}/raw", server.base),
            };
            Fixture {
                server,
                endpoints,
                binary,
                license,
            }
        }

        fn plain() -> Fixture {
            Fixture::start()
        }
    }

    #[cfg(unix)]
    fn mode(path: &Path) -> u32 {
        use std::os::unix::fs::PermissionsExt;
        std::fs::metadata(path).unwrap().permissions().mode() & 0o777
    }

    #[test]
    fn tag_shape_matches_version_tags_only() {
        assert!(valid_tag("v0.99.1"));
        assert!(valid_tag("v10.20.30"));
        for bad in [
            "",
            "v",
            "0.99.1",
            "vv0.99.1",
            "v0.99",
            "v0.99.1.2",
            "v0.99.x",
            "v0.99.1 ",
            " v0.99.1",
            "vyesterday",
            "v１.2.3",
        ] {
            assert!(!valid_tag(bad), "{bad:?}");
        }
    }

    #[test]
    fn checksums_take_the_last_match_and_ignore_odd_lines() {
        let text = "aaa  other\nzzz  tea-1-linux-amd64\nwww  tea-1-linux-amd64\nlone\n\n";
        assert_eq!(
            checksums_for(text, "tea-1-linux-amd64"),
            Some("www".to_string())
        );
        assert_eq!(checksums_for(text, "missing"), None);
        assert_eq!(checksums_for("not a pair of fields here", "x"), None);
    }

    #[test]
    fn x86_64_stages_exact_bytes_license_modes_and_message() {
        let fixture = Fixture::plain();
        let scratch = TempDir::new("tea-ok");
        let out = scratch.path.join("output");
        let message = fetch("x86_64", &out, &fixture.endpoints).unwrap();
        assert_eq!(
            message,
            format!(
                "Upstream Tea {VERSION} (x86_64) staged at {}",
                out.display()
            )
        );
        assert_eq!(std::fs::read(out.join("bin/tea")).unwrap(), fixture.binary);
        assert_eq!(
            std::fs::read(out.join("licenses/tea/LICENSE")).unwrap(),
            fixture.license
        );
        #[cfg(unix)]
        {
            assert_eq!(mode(&out.join("bin/tea")), 0o755);
            assert_eq!(mode(&out.join("licenses/tea/LICENSE")), 0o644);
        }
        for observed in fixture.server.seen() {
            assert_eq!(observed.user_agent, "SodaOS-build");
        }
        assert_eq!(fixture.server.seen().len(), 4);
    }

    #[test]
    fn unsupported_architecture_is_refused_without_fetch() {
        let fixture = Fixture::plain();
        let scratch = TempDir::new("tea-arch");
        let out = scratch.path.join("output");
        assert_eq!(
            fetch("aarch64", &out, &fixture.endpoints).unwrap_err(),
            "Tea staging supports x86_64 only"
        );
        assert!(fixture.server.seen().is_empty());
        assert!(!out.exists());
    }

    #[test]
    fn corrupt_download_leaves_no_stage() {
        // A corrupted binary served under the good pin.
        let binary = b"corrupt".to_vec();
        let mut raw: HashMap<String, (u16, Vec<u8>)> = HashMap::new();
        let good = elf_body(62);
        let sums = format!(
            "{}  tea-{VERSION}-linux-amd64\n",
            crate::fetch::sha256_hex(&good)
        );
        raw.insert(
            "/api/releases/latest".to_string(),
            (200, format!(r#"{{"tag_name":"{TAG}"}}"#).into_bytes()),
        );
        raw.insert(
            format!("/{VERSION}/checksums.txt"),
            (200, sums.into_bytes()),
        );
        raw.insert(
            format!("/{VERSION}/tea-{VERSION}-linux-amd64"),
            (200, binary),
        );
        raw.insert(
            format!("/raw/{TAG}/LICENSE"),
            (200, b"fixture license\n".to_vec()),
        );
        let server = Server::start(raw);
        let endpoints = Endpoints {
            releases_api: format!("{}/api/releases/latest", server.base),
            dl_base: server.base.clone(),
            license_base: format!("{}/raw", server.base),
        };
        let scratch = TempDir::new("tea-corrupt");
        let out = scratch.path.join("output");
        assert_eq!(
            fetch("x86_64", &out, &endpoints).unwrap_err(),
            "Tea binary checksum mismatch"
        );
        assert!(!out.exists());
        assert!(server.seen().len() >= 3);
    }

    #[test]
    fn wrong_architecture_binary_leaves_no_stage() {
        let binary = elf_body(183);
        let mut raw: HashMap<String, (u16, Vec<u8>)> = HashMap::new();
        let sums = format!(
            "{}  tea-{VERSION}-linux-amd64\n",
            crate::fetch::sha256_hex(&binary)
        );
        raw.insert(
            "/api/releases/latest".to_string(),
            (200, format!(r#"{{"tag_name":"{TAG}"}}"#).into_bytes()),
        );
        raw.insert(
            format!("/{VERSION}/checksums.txt"),
            (200, sums.into_bytes()),
        );
        raw.insert(
            format!("/{VERSION}/tea-{VERSION}-linux-amd64"),
            (200, binary),
        );
        raw.insert(
            format!("/raw/{TAG}/LICENSE"),
            (200, b"fixture license\n".to_vec()),
        );
        let server = Server::start(raw);
        let endpoints = Endpoints {
            releases_api: format!("{}/api/releases/latest", server.base),
            dl_base: server.base.clone(),
            license_base: format!("{}/raw", server.base),
        };
        let scratch = TempDir::new("tea-machine");
        let out = scratch.path.join("output");
        assert_eq!(
            fetch("x86_64", &out, &endpoints).unwrap_err(),
            "Tea binary is not ELF64 for the requested architecture"
        );
        assert!(!out.exists());
    }

    #[test]
    fn non_version_tag_leaves_no_stage() {
        let mut raw: HashMap<String, (u16, Vec<u8>)> = HashMap::new();
        raw.insert(
            "/api/releases/latest".to_string(),
            (200, br#"{"tag_name":"yesterday"}"#.to_vec()),
        );
        let server = Server::start(raw);
        let endpoints = Endpoints {
            releases_api: format!("{}/api/releases/latest", server.base),
            dl_base: server.base.clone(),
            license_base: format!("{}/raw", server.base),
        };
        let scratch = TempDir::new("tea-tag");
        let out = scratch.path.join("output");
        assert_eq!(
            fetch("x86_64", &out, &endpoints).unwrap_err(),
            "Tea latest release is not a version tag"
        );
        assert!(!out.exists());
        assert_eq!(server.seen().len(), 1);
    }

    #[test]
    fn empty_license_leaves_no_stage() {
        let binary = elf_body(62);
        let mut raw: HashMap<String, (u16, Vec<u8>)> = HashMap::new();
        let sums = format!(
            "{}  tea-{VERSION}-linux-amd64\n",
            crate::fetch::sha256_hex(&binary)
        );
        raw.insert(
            "/api/releases/latest".to_string(),
            (200, format!(r#"{{"tag_name":"{TAG}"}}"#).into_bytes()),
        );
        raw.insert(
            format!("/{VERSION}/checksums.txt"),
            (200, sums.into_bytes()),
        );
        raw.insert(
            format!("/{VERSION}/tea-{VERSION}-linux-amd64"),
            (200, binary),
        );
        raw.insert(format!("/raw/{TAG}/LICENSE"), (200, Vec::new()));
        let server = Server::start(raw);
        let endpoints = Endpoints {
            releases_api: format!("{}/api/releases/latest", server.base),
            dl_base: server.base.clone(),
            license_base: format!("{}/raw", server.base),
        };
        let scratch = TempDir::new("tea-license");
        let out = scratch.path.join("output");
        assert_eq!(
            fetch("x86_64", &out, &endpoints).unwrap_err(),
            "Tea license download is empty"
        );
        assert!(!out.exists());
    }

    #[test]
    fn checksums_without_the_archive_leave_no_stage() {
        let mut raw: HashMap<String, (u16, Vec<u8>)> = HashMap::new();
        raw.insert(
            "/api/releases/latest".to_string(),
            (200, format!(r#"{{"tag_name":"{TAG}"}}"#).into_bytes()),
        );
        raw.insert(
            format!("/{VERSION}/checksums.txt"),
            (200, b"aaa  something-else\n".to_vec()),
        );
        let server = Server::start(raw);
        let endpoints = Endpoints {
            releases_api: format!("{}/api/releases/latest", server.base),
            dl_base: server.base.clone(),
            license_base: format!("{}/raw", server.base),
        };
        let scratch = TempDir::new("tea-sums");
        let out = scratch.path.join("output");
        assert_eq!(
            fetch("x86_64", &out, &endpoints).unwrap_err(),
            "Tea checksums omit the requested archive"
        );
        assert!(!out.exists());
    }

    #[test]
    fn shared_output_dir_stages_alongside_existing_files() {
        let fixture = Fixture::plain();
        let scratch = TempDir::new("tea-shared");
        let out = scratch.path.join("output");
        std::fs::create_dir_all(out.join("bin")).unwrap();
        std::fs::write(out.join("bin/soda-host"), b"keep").unwrap();
        fetch("x86_64", &out, &fixture.endpoints).unwrap();
        assert_eq!(std::fs::read(out.join("bin/tea")).unwrap(), fixture.binary);
        assert_eq!(
            std::fs::read(out.join("licenses/tea/LICENSE")).unwrap(),
            fixture.license
        );
        assert_eq!(std::fs::read(out.join("bin/soda-host")).unwrap(), b"keep");
    }

    #[test]
    fn existing_tea_outputs_are_refused_without_fetch() {
        for (index, sentinel) in ["bin/tea", "licenses/tea/LICENSE"].iter().enumerate() {
            let fixture = Fixture::plain();
            let scratch = TempDir::new("tea-occupied");
            let out = scratch.path.join(format!("occupied-{index}"));
            let target = out.join(sentinel);
            std::fs::create_dir_all(target.parent().unwrap()).unwrap();
            std::fs::write(&target, b"keep").unwrap();
            assert_eq!(
                fetch("x86_64", &out, &fixture.endpoints).unwrap_err(),
                "Tea output already exists; select a fresh output directory"
            );
            assert!(fixture.server.seen().is_empty());
            assert_eq!(std::fs::read(&target).unwrap(), b"keep");
        }
    }

    #[test]
    fn dangling_symlink_outputs_are_refused_without_fetch() {
        let fixture = Fixture::plain();
        let scratch = TempDir::new("tea-link");
        let out = scratch.path.join("output");
        std::fs::create_dir_all(out.join("bin")).unwrap();
        #[cfg(unix)]
        std::os::unix::fs::symlink("nowhere", out.join("bin/tea")).unwrap();
        assert_eq!(
            fetch("x86_64", &out, &fixture.endpoints).unwrap_err(),
            "Tea output already exists; select a fresh output directory"
        );
        assert!(fixture.server.seen().is_empty());
    }

    #[test]
    fn http_errors_fail_without_a_stage() {
        let mut raw: HashMap<String, (u16, Vec<u8>)> = HashMap::new();
        raw.insert("/api/releases/latest".to_string(), (404, Vec::new()));
        let server = Server::start(raw);
        let endpoints = Endpoints {
            releases_api: format!("{}/api/releases/latest", server.base),
            dl_base: server.base.clone(),
            license_base: format!("{}/raw", server.base),
        };
        let scratch = TempDir::new("tea-404");
        let out = scratch.path.join("output");
        let err = fetch("x86_64", &out, &endpoints).unwrap_err();
        assert!(err.contains("404"), "{err}");
        assert!(!out.exists());
    }

    #[test]
    fn default_out_is_repo_shaped() {
        let out = default_out("x86_64").unwrap();
        assert!(out.is_absolute());
        assert!(
            out.ends_with(".artifacts/native/x86_64/project-tools"),
            "{}",
            out.display()
        );
    }
}
