//! Private browser setup: address selection, the Forgejo operator token
//! flow through the native setup/activation binaries, and read-only
//! configured-access guidance.

use crate::command::{failure_summary, Runner};
use crate::console::Console;
use crate::errors::{self, Error};
use crate::fmtx::Arg;
use crate::jsongo::{parse, Soft};
use crate::netip;
use crate::signal::Ctx;

const LOCAL_CA_PATH: &str = "/var/lib/soda/proxy/caddy/pki/authorities/local/root.crt";
/// `platform.Sbin`: native Soda binaries live here.
const SBIN: &str = "/usr/bin";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SetupAddress {
    pub interface: String,
    pub address: String,
}

fn skip_setup_interface(name: &str, flags: &[String]) -> bool {
    let mut up = false;
    for flag in flags {
        up = up || flag == "UP";
    }
    !up || name == "soda0" || name.starts_with("podman") || name.starts_with("veth")
}

fn append_setup_address(
    choices: &mut Vec<SetupAddress>,
    seen: &mut std::collections::HashSet<String>,
    excluded: &mut Vec<String>,
    name: &str,
    local: &str,
    scope: &str,
) {
    if private_setup_origin(local).is_ok() && scope == "global" && !seen.contains(local) {
        choices.push(SetupAddress { interface: name.to_string(), address: local.to_string() });
        seen.insert(local.to_string());
        return;
    }
    if excluded.len() < 8 {
        let reason = if scope != "global" {
            format!("scope {scope}, not global")
        } else if seen.contains(local) {
            "duplicate address".to_string()
        } else {
            "not a private LAN or Tailscale address".to_string()
        };
        excluded.push(format!("{name} {local} ({reason})"));
    }
}

pub fn private_setup_origin(value: &str) -> Result<String, Error> {
    let cgnat = netip::parse_prefix("100.64.0.0/10").expect("CGNAT prefix is valid");
    let address = netip::parse_addr(value).map_err(|_| Error::msg("select an assigned private LAN or Tailscale address"))?;
    if !address.zone().is_empty()
        || address.is4_in6()
        || (!address.is_private() && !cgnat.contains(&address))
    {
        return Err(Error::msg("select an assigned private LAN or Tailscale address"));
    }
    let mut host = String::from_utf8_lossy(&address.to_string_go()).into_owned();
    if address.is6() {
        host = format!("[{host}]");
    }
    Ok(format!("https://{host}"))
}

fn setup_addresses(data: &[u8]) -> Result<Vec<SetupAddress>, Error> {
    let value = parse(data).map_err(|_| Error::msg("cannot inspect private setup addresses"))?;
    let interfaces = match &value {
        soda_json::JsonValue::Array(items) => items,
        _ => return Err(Error::msg("cannot inspect private setup addresses")),
    };
    let mut choices = Vec::new();
    let mut excluded = Vec::new();
    let mut seen = std::collections::HashSet::new();
    for item in interfaces {
        let network = Soft::new(item).map_err(|_| Error::msg("cannot inspect private setup addresses"))?;
        let name = network.string("ifname").map_err(|_| Error::msg("cannot inspect private setup addresses"))?.unwrap_or_default();
        let flags: Vec<String> = match network.array("flags").map_err(|_| Error::msg("cannot inspect private setup addresses"))? {
            None => Vec::new(),
            Some(items) => {
                let mut flags = Vec::new();
                for item in items {
                    match item {
                        soda_json::JsonValue::Str(flag) => flags.push(flag.clone()),
                        _ => return Err(Error::msg("cannot inspect private setup addresses")),
                    }
                }
                flags
            }
        };
        if skip_setup_interface(&name, &flags) {
            continue;
        }
        let addr_info = network.array("addr_info").map_err(|_| Error::msg("cannot inspect private setup addresses"))?.unwrap_or(&[]);
        for entry in addr_info {
            let address = Soft::new(entry).map_err(|_| Error::msg("cannot inspect private setup addresses"))?;
            let local = address.string("local").map_err(|_| Error::msg("cannot inspect private setup addresses"))?.unwrap_or_default();
            let scope = address.string("scope").map_err(|_| Error::msg("cannot inspect private setup addresses"))?.unwrap_or_default();
            append_setup_address(&mut choices, &mut seen, &mut excluded, &name, &local, &scope);
        }
    }
    if choices.is_empty() {
        // Name what the filter rejected so the operator can compare with the
        // unfiltered live addresses shown during disk installation.
        if excluded.is_empty() {
            return Err(Error::msg("no private setup address is available; configure networking first"));
        }
        return Err(Error::msg(format!(
            "no private setup address is available; configure networking first (observed but unusable: {})",
            excluded.join("; ")
        )));
    }
    Ok(choices)
}

// configureInstall uses the existing native Forgejo installer and setup command.
// Soda does not create a second account/password authority or expose the unfinished
// Forgejo installer. It runs in any interactive operator terminal, local or SSH,
// so the token can be typed or pasted; no SSH session is required.
pub fn configure_install(ctx: &Ctx, console: &Console, run: &dyn Runner) -> Result<(), Error> {
    configure_private_install(ctx, console, run, "/etc/soda", "/run", LOCAL_CA_PATH)
}

fn check_preexisting_install(root: &str) -> Result<bool, Error> {
    if crate::execute::read_regular(&format!("{root}/installed"), 256).is_err() {
        return Err(Error::msg("install the included Soda components before configuring browser access"));
    }
    match std::fs::symlink_metadata(format!("{root}/activated")) {
        Ok(_) => return Ok(true),
        Err(err) if err.kind() != std::io::ErrorKind::NotFound => {
            return Err(Error::msg("cannot inspect existing activation"))
        }
        Err(_) => {}
    }
    // Partial states refuse automatic replay, but each names its explicit
    // operator recovery instead of stranding a failed attempt with no retry.
    if !matches!(
        std::fs::symlink_metadata(format!("{root}/dashboard.json")),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound
    ) {
        return Err(Error::msg("operator configuration already exists; complete activation with the installed soda-activate command, or inspect the existing files before any manual maintenance; configure will not replay setup"));
    }
    if !matches!(
        std::fs::symlink_metadata(format!("{root}/setup-started")),
        Err(err) if err.kind() == std::io::ErrorKind::NotFound
    ) {
        return Err(Error::msg(format!("a previous setup attempt reserved {root}/setup-started but wrote no configuration; confirm no setup is running, remove only that reservation file, then rerun configure")));
    }
    Ok(false)
}

fn query_setup_addresses(ctx: &Ctx, run: &dyn Runner) -> Result<Vec<SetupAddress>, Error> {
    let data = run.run(ctx, "ip", &["-json".to_string(), "address".to_string(), "show".to_string(), "up".to_string()], None)
        .map_err(|_| Error::msg("cannot inspect network addresses"))?;
    setup_addresses(&data)
}

fn prompt_setup_address(ctx: &Ctx, console: &Console, choices: &[SetupAddress]) -> Result<SetupAddress, Error> {
    console.page("Private browser setup");
    console.print("Use this SSH terminal to paste the Forgejo token when asked; input will be hidden.", &[]);
    console.print("Select the appliance address your laptop can reach. No domain is needed.", &[]);
    for (i, choice) in choices.iter().enumerate() {
        console.print(
            "%d. %s on %q",
            &[Arg::Int((i + 1) as i64), Arg::Str(&choice.address), Arg::Str(&choice.interface)],
        );
    }
    loop {
        let answer = console.ask(ctx, "Address number, or cancel")?;
        if answer == "cancel" {
            return Err(Error::msg("browser setup cancelled"));
        }
        if let Ok(i) = answer.parse::<i64>() {
            if i > 0 && (i as usize) <= choices.len() {
                return Ok(choices[(i as usize) - 1].clone());
            }
        }
        console.print("Choose one of the listed address numbers.", &[]);
    }
}

fn valid_operator_token(token: &[u8]) -> bool {
    if token.is_empty() || token.iter().any(|b| matches!(b, b'\r' | b'\n' | b'\0')) {
        return false;
    }
    // Go compares against Unicode whitespace trimming; lossy decoding keeps
    // invalid bytes visible (never whitespace), exactly like Go's RuneError.
    let text = String::from_utf8_lossy(token);
    text.as_ref() == text.trim()
}

fn prompt_operator_token(
    ctx: &Ctx,
    console: &Console,
    address: &str,
    origin: &str,
) -> Result<Vec<u8>, Error> {
    console.print("The final Soda address will be %s", &[Arg::Str(origin)]);
    console.print(
        "Use a stable address or DHCP reservation. Changing it later needs explicit configuration maintenance.",
        &[],
    );
    console.print(
        "If you have no SSH key access yet, cancel and run %s enroll-key at the local console.",
        &[Arg::Str(crate::candidate::CANDIDATE_INSTALLER_BINARY)],
    );
    console.print(
        "From your laptop, connect with an SSH tunnel to the native Forgejo installer:",
        &[],
    );
    console.print("ssh -L 33000:127.0.0.1:3000 root@%s", &[Arg::Str(address)]);
    console.print(
        "Open http://localhost:33000 and complete Forgejo's own installation and administrator account setup.",
        &[],
    );
    console.print(
        "Keep its localhost browser URL for this bootstrap; activation below sets the final private URL.",
        &[],
    );
    console.print(
        "In Forgejo Settings > Applications, create the operator token described in the operator setup guide.",
        &[],
    );
    console.print("Required scope: read:user. No admin or repository scope is needed. Paste it here through the SSH terminal; input is hidden.", &[]);
    console.print(
        "Caddy will issue local HTTPS certificates. You will explicitly trust its public root certificate on your laptop.",
        &[],
    );
    let answer = console.ask(ctx, "When Forgejo setup is complete, type CONFIGURE SODA; anything else cancels")?;
    if answer != "CONFIGURE SODA" {
        return Err(Error::msg("browser setup cancelled; no Soda configuration written"));
    }
    let token = console.secret(ctx, "Operator Forgejo token")?;
    if !valid_operator_token(&token) {
        return Err(Error::msg("operator token is empty or malformed"));
    }
    Ok(token)
}

fn verify_address_still_assigned(
    ctx: &Ctx,
    run: &dyn Runner,
    selected: &SetupAddress,
) -> Result<(), Error> {
    let current =
        query_setup_addresses(ctx, run).map_err(|_| Error::msg("cannot recheck the selected address"))?;
    if current.iter().any(|choice| choice == selected) {
        return Ok(());
    }
    Err(Error::msg("selected network address changed; restart browser setup"))
}

fn make_setup_workdir(temporary: &str) -> Result<String, Error> {
    for _ in 0..100 {
        let path = format!("{temporary}/soda-setup-{}", crate::enroll::keys::random_hex(8)?);
        use std::os::unix::fs::DirBuilderExt;
        match std::fs::DirBuilder::new().mode(0o700).create(&path) {
            Ok(()) => return Ok(path),
            Err(err) if err.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(err) => return Err(errors::path_error("mkdir", &path, err)),
        }
    }
    Err(errors::path_error(
        "mkdir",
        temporary,
        std::io::Error::from_raw_os_error(libc::EEXIST),
    ))
}

fn execute_setup_and_activation(
    ctx: &Ctx,
    run: &dyn Runner,
    root: &str,
    temporary: &str,
    origin: &str,
    token: &[u8],
    selected: &SetupAddress,
) -> Result<(), Error> {
    let work = make_setup_workdir(temporary)?;
    // The operator token must not outlive this attempt, including failures.
    struct WorkDir(String);
    impl Drop for WorkDir {
        fn drop(&mut self) {
            let _ = std::fs::remove_dir_all(&self.0);
        }
    }
    let _work = WorkDir(work.clone());
    let token_path = format!("{work}/operator-token");
    let mut token_data = token.to_vec();
    token_data.push(b'\n');
    write_setup_file(&token_path, &token_data)?;
    let started_path = format!("{root}/setup-started");
    write_setup_file(&started_path, format!("{origin}\n").as_bytes())
        .map_err(|_| Error::msg("cannot reserve operator setup; no native API request made"))?;
    if let Err(err) = run.run(
        ctx,
        &format!("{SBIN}/soda-setup"),
        &[
            "--forgejo-url".to_string(),
            origin.to_string(),
            "--token-file".to_string(),
            token_path,
            "--out".to_string(),
            format!("{root}/dashboard.json"),
        ],
        None,
    ) {
        // Roll back only this attempt's reservation: without dashboard.json
        // nothing references the marker or the orphaned grant key, so a plain
        // rerun is safe. A written dashboard.json is always preserved.
        let summary = failure_summary(&err);
        if matches!(
            std::fs::symlink_metadata(format!("{root}/dashboard.json")),
            Err(stat_err) if stat_err.kind() == std::io::ErrorKind::NotFound
        ) {
            let _ = std::fs::remove_file(&started_path);
            let _ = std::fs::remove_file(format!("{root}/grant-key"));
            return Err(Error::msg(format!("operator setup failed; this attempt wrote no configuration and its reservation was removed, so rerunning configure is safe after addressing the cause. {summary}")));
        }
        return Err(Error::msg(format!(
            "operator setup failed; inspect the existing configuration before retrying. {summary}"
        )));
    }
    if let Err(err) = run.run(
        ctx,
        &format!("{SBIN}/soda-activate"),
        &["--bind-ip".to_string(), selected.address.clone(), "--local-tls".to_string()],
        None,
    ) {
        return Err(Error::msg(format!(
            "private activation failed; preserve the existing configuration for inspection. {}",
            failure_summary(&err)
        )));
    }
    Ok(())
}

fn configure_private_install(
    ctx: &Ctx,
    console: &Console,
    run: &dyn Runner,
    root: &str,
    temporary: &str,
    ca_path: &str,
) -> Result<(), Error> {
    if check_preexisting_install(root)? {
        return configured_access(ctx, console, root, ca_path, run);
    }
    let choices = query_setup_addresses(ctx, run)?;
    let selected = prompt_setup_address(ctx, console, &choices)?;
    let origin = private_setup_origin(&selected.address).unwrap_or_default();
    let token = prompt_operator_token(ctx, console, &selected.address, &origin)?;
    verify_address_still_assigned(ctx, run, &selected)?;
    execute_setup_and_activation(ctx, run, root, temporary, &origin, &token, &selected)?;
    configured_access(ctx, console, root, ca_path, run)
}

fn write_setup_file(path: &str, data: &[u8]) -> Result<(), Error> {
    use std::io::Write;
    use std::os::unix::fs::OpenOptionsExt;
    // Single write, byte count ignored, close still attempted: write error
    // wins over close error, like Go's writeSetupFile.
    let mut file = std::fs::OpenOptions::new()
        .write(true)
        .create_new(true)
        .mode(0o600)
        .open(path)
        .map_err(|e| errors::path_error("open", path, e))?;
    let write_err = file.write(data).err().map(|e| errors::path_error("write", path, e));
    use std::os::unix::io::IntoRawFd;
    let close_err = if unsafe { libc::close(file.into_raw_fd()) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        Some(errors::path_error("close", path, std::io::Error::from_raw_os_error(errno)))
    } else {
        None
    };
    if let Some(err) = write_err {
        return Err(err);
    }
    if let Some(err) = close_err {
        return Err(err);
    }
    Ok(())
}

fn valid_installed_https_origin(origin: &crate::urlx::Url, raw: &str) -> bool {
    if origin.scheme != "https"
        || origin.user_present
        || !origin.raw_query.is_empty()
        || !origin.fragment.is_empty()
    {
        return false;
    }
    if !origin.path.is_empty() && origin.path != b"/" {
        return false;
    }
    !raw.bytes().any(|b| matches!(b, b'\r' | b'\n' | b'\0'))
}

fn installed_forgejo_origin(root: &str) -> Result<(crate::urlx::Url, String), Error> {
    let data = crate::execute::read_regular(&format!("{root}/dashboard.json"), 65536)
        .map_err(|_| Error::msg("cannot read installed browser address"))?;
    let value = parse(&data).map_err(|_| Error::msg("invalid installed browser configuration"))?;
    let config = Soft::new(&value).map_err(|_| Error::msg("invalid installed browser configuration"))?;
    let url = config
        .string("forgejo_url")
        .map_err(|_| Error::msg("invalid installed browser configuration"))?
        .unwrap_or_default();
    let origin =
        crate::urlx::parse(&url).map_err(|_| Error::msg("invalid installed HTTPS address"))?;
    if !valid_installed_https_origin(&origin, &url) {
        return Err(Error::msg("invalid installed HTTPS address"));
    }
    Ok((origin, url))
}

fn uses_internal_tls(root: &str) -> Result<bool, Error> {
    let proxy = crate::execute::read_regular(&format!("{root}/proxy.env"), 65536)
        .map_err(|_| Error::msg("cannot inspect the installed TLS mode"))?;
    Ok(proxy.split(|b| *b == b'\n').any(|line| line == b"SODA_TLS=internal"))
}

fn confirm_active_browser_units(ctx: &Ctx, run: &dyn Runner) -> Result<(), Error> {
    for unit in ["forgejo.service", "soda-dashboard.service", "soda-proxy.service"] {
        if run.run(ctx, "systemctl", &["is-active".to_string(), "--quiet".to_string(), unit.to_string()], None).is_err() {
            return Err(Error::msg(format!("private activation was requested, but {unit} is not confirmed active; inspect its native service state, then rerun configure for read-only guidance; setup was not replayed")));
        }
    }
    Ok(())
}

fn print_local_ca_guidance(
    console: &Console,
    origin: &crate::urlx::Url,
    address: &str,
    ca_path: &str,
) -> Result<(), Error> {
    let certificate = match crate::execute::read_regular(ca_path, 16384) {
        Ok(certificate) => certificate,
        Err(_) => {
            // Active units without trust material are not a ready browser
            // address: say so before any Open guidance, not after it.
            console.print("The browser services report active, but the local CA certificate is not available yet; do not open the browser address until the trust material below exists.", &[]);
            console.print("Inspect soda-proxy.service, then run configure again to show the trust instructions. Existing setup will not be replayed.", &[]);
            return Ok(());
        }
    };
    let fingerprint = local_ca_fingerprint(&certificate)?;
    console.print("The browser services report active. Open %s only after completing the client trust below; browser login still needs verification.", &[Arg::Str(address)]);
    console.print("Local CA certificate SHA-256: %s", &[Arg::Str(&fingerprint)]);
    console.print("Copy only the public root.crt file over your verified SSH connection:", &[]);
    let host = String::from_utf8_lossy(&origin.host);
    console.print("scp root@%s:%s ./soda-local-ca.crt", &[Arg::Str(&host), Arg::Str(ca_path)]);
    console.print("Compare its certificate fingerprint, then trust it in your laptop/browser certificate settings. Never copy the CA private key.", &[]);
    Ok(())
}

fn configured_access(
    ctx: &Ctx,
    console: &Console,
    root: &str,
    ca_path: &str,
    run: &dyn Runner,
) -> Result<(), Error> {
    let (origin, address) = installed_forgejo_origin(root)?;
    let local = uses_internal_tls(root)?;
    if local {
        let hostname = String::from_utf8_lossy(&origin.hostname());
        let expected = private_setup_origin(&hostname).unwrap_or_default();
        if address.strip_suffix('/').unwrap_or(&address) != expected {
            return Err(Error::msg("local TLS must use the selected private IP origin"));
        }
    }
    confirm_active_browser_units(ctx, run)?;
    if local {
        print_local_ca_guidance(console, &origin, &address, ca_path)?;
    } else {
        console.print("The browser services report active. Open %s after setting up client trust for your supplied certificate; browser login still needs verification.", &[Arg::Str(&address)]);
    }
    console.print("Sign in to native Forgejo, create or choose a repository, and open Sodaspaces to create and join its development environment.", &[]);
    console.print("Verify a browser terminal in that project. Opening this setup screen is not a completed project/access test.", &[]);
    Ok(())
}

fn local_ca_fingerprint(data: &[u8]) -> Result<String, Error> {
    let (block, rest) = crate::pemx::decode(data);
    let block = match block {
        Some(block) if block.der_type == "CERTIFICATE" => block,
        _ => return Err(Error::msg("expected one public CA certificate")),
    };
    if !String::from_utf8_lossy(rest).trim().is_empty() {
        return Err(Error::msg("expected one public CA certificate"));
    }
    let certificate = crate::x509::parse_certificate(&block.bytes)
        .map_err(|_| Error::msg("invalid local CA certificate"))?;
    if !certificate.is_ca
        || !certificate.basic_constraints_valid
        || crate::x509::check_signature_from(&certificate, &certificate).is_err()
    {
        return Err(Error::msg("invalid local CA certificate"));
    }
    Ok(crate::buildx::sha256_hex(&certificate.raw))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::command::FnRunner;
    use crate::enroll::tests::{drain_available, open_test_pty, temp_dir, ENV_LOCK};

    #[test]
    fn private_setup_addresses() {
        for (address, expected) in [
            ("192.168.2.100", "https://192.168.2.100"),
            ("100.90.1.2", "https://100.90.1.2"),
            ("fd00::123", "https://[fd00::123]"),
        ] {
            assert_eq!(private_setup_origin(address).unwrap(), expected, "{address:?}");
        }
        for address in [
            "127.0.0.1",
            "::1",
            "0.0.0.0",
            "8.8.8.8",
            "169.254.1.2",
            "fe80::1%eth0",
            "192.168.1.2;reboot",
            "soda.example.test",
            "::ffff:192.168.1.2",
        ] {
            assert!(private_setup_origin(address).is_err(), "{address:?}");
        }
        let data = br#"[
		{"ifname":"eth0","flags":["UP"],"addr_info":[{"local":"192.168.1.5","scope":"global"},{"local":"8.8.8.8","scope":"global"}]},
		{"ifname":"eth1","flags":[],"addr_info":[{"local":"10.0.0.2","scope":"global"}]},
		{"ifname":"soda0","flags":["UP"],"addr_info":[{"local":"10.89.0.1","scope":"global"}]},
		{"ifname":"lo","flags":["UP"],"addr_info":[{"local":"127.0.0.1","scope":"host"}]}
	]"#;
        let addresses = setup_addresses(data).unwrap();
        assert_eq!(
            addresses,
            vec![SetupAddress { interface: "eth0".to_string(), address: "192.168.1.5".to_string() }]
        );
        for data in ["not json", "[]"] {
            assert!(setup_addresses(data.as_bytes()).is_err(), "{data:?}");
        }
    }

    fn tlv(tag: u8, contents: &[u8]) -> Vec<u8> {
        let mut out = vec![tag];
        if contents.len() < 128 {
            out.push(contents.len() as u8);
        } else {
            let mut len = contents.len();
            let mut bytes = Vec::new();
            while len > 0 {
                bytes.push((len & 0xff) as u8);
                len >>= 8;
            }
            out.push(0x80 | bytes.len() as u8);
            bytes.reverse();
            out.extend_from_slice(&bytes);
        }
        out.extend_from_slice(contents);
        out
    }

    fn ca_cert_pem(is_ca: bool) -> Vec<u8> {
        use ed25519_dalek::Signer as _;
        let signing = ed25519_dalek::SigningKey::from_bytes(&[
            0x9D, 0x61, 0xB1, 0x9D, 0xEF, 0xFD, 0x5A, 0x60, 0xBA, 0x84, 0x4A, 0xF4, 0x92,
            0x2E, 0xC4, 0x44, 0x48, 0xC8, 0x58, 0x07, 0x31, 0x11, 0xED, 0xD3, 0xAD, 0x45,
            0x8B, 0x22, 0x7E, 0x4E, 0x4B, 0x63,
        ]);
        let seq_of = |parts: &[Vec<u8>]| {
            let mut contents = Vec::new();
            for part in parts {
                contents.extend_from_slice(part);
            }
            tlv(0x30, &contents)
        };
        let ai = tlv(0x30, &tlv(0x06, &[0x2B, 0x65, 0x70]));
        let validity = seq_of(&[tlv(0x17, b"700101000000Z"), tlv(0x17, b"700102000000Z")]);
        let spki = seq_of(&[
            ai.clone(),
            tlv(0x03, &[&[0x00], signing.verifying_key().as_bytes().as_slice()].concat()),
        ]);
        let bc = tlv(0x30, &tlv(0x01, &[if is_ca { 0xFF } else { 0x00 }]));
        let ext = seq_of(&[tlv(0x06, &[0x55, 0x1D, 0x13]), tlv(0x01, &[0xFF]), tlv(0x04, &bc)]);
        let tbs = seq_of(&[
            tlv(0xA0, &tlv(0x02, &[0x02])),
            tlv(0x02, &[0x01]),
            ai.clone(),
            tlv(0x30, &[]),
            validity,
            tlv(0x30, &[]),
            spki,
            tlv(0xA3, &seq_of(&[ext])),
        ]);
        let sig = signing.sign(&tbs);
        let cert = seq_of(&[tbs, ai, tlv(0x03, &[&[0x00], sig.to_bytes().as_slice()].concat())]);
        let b64 = crate::sshkey::b64_encode(&cert);
        let mut pem = b"-----BEGIN CERTIFICATE-----\n".to_vec();
        for chunk in b64.as_bytes().chunks(64) {
            pem.extend_from_slice(chunk);
            pem.push(b'\n');
        }
        pem.extend_from_slice(b"-----END CERTIFICATE-----\n");
        pem
    }

    #[test]
    fn local_certificate_export_requires_public_ca() {
        let encoded = ca_cert_pem(true);
        let fingerprint = local_ca_fingerprint(&encoded).unwrap();
        assert_eq!(fingerprint.len(), 64);
        assert!(fingerprint.bytes().all(|b| b.is_ascii_hexdigit()));
        assert!(local_ca_fingerprint(&ca_cert_pem(false)).is_err());
        let mut trailing = encoded.clone();
        trailing.extend_from_slice(b"private trailing data");
        assert!(local_ca_fingerprint(&trailing).is_err());
        let key = b"-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n";
        assert!(local_ca_fingerprint(key).is_err());
    }

    fn null_console() -> Console {
        Console::from_file("/dev/null", std::fs::File::open("/dev/null").unwrap())
    }

    #[test]
    fn private_setup_does_not_replay_existing_state() {
        for state in ["dashboard.json", "setup-started", "activated"] {
            let root = temp_dir();
            std::fs::write(format!("{}/installed", root.path), b"").unwrap();
            std::fs::write(format!("{}/{}", root.path, state), b"").unwrap();
            let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
                panic!("existing state caused a native command or another setup attempt")
            });
            let (ctx, _flag) = Ctx::test();
            // Empty existing config fails read-only, without repeating setup.
            assert!(
                configure_private_install(&ctx, &null_console(), &run, &root.path, &root.path, "missing-ca").is_err(),
                "{state}"
            );
        }
    }

    #[test]
    fn local_trust_guidance_rejects_untrusted_destination() {
        for origin in [
            "https://8.8.8.8",
            "https://soda.example.test",
            "https://192.168.1.2:444",
            "https://192.168.1.2/path",
            "https://root@192.168.1.2",
            "https://192.168.1.2?argument",
        ] {
            let root = temp_dir();
            std::fs::write(
                format!("{}/dashboard.json", root.path),
                format!(r#"{{"forgejo_url":"{origin}"}}"#),
            )
            .unwrap();
            std::fs::write(format!("{}/proxy.env", root.path), b"SODA_TLS=internal\n").unwrap();
            let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
                panic!("invalid origin reached native service inspection")
            });
            let (ctx, _flag) = Ctx::test();
            assert!(
                configured_access(&ctx, &null_console(), &root.path, "missing-ca", &run).is_err(),
                "{origin}"
            );
        }
    }

    #[test]
    fn configure_runs_without_laptop_terminal() {
        let _guard = ENV_LOCK.lock().unwrap();
        std::env::set_var("SSH_CONNECTION", "");
        std::env::set_var("SSH_TTY", "");
        let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            panic!("local configure reached native operations")
        });
        let (ctx, _flag) = Ctx::test();
        let err = configure_install(&ctx, &null_console(), &run).unwrap_err();
        assert!(err.to_string().contains("install the included Soda components"), "{err}");
        std::env::remove_var("SSH_CONNECTION");
        std::env::remove_var("SSH_TTY");
    }

    #[test]
    fn configured_guidance_reports_inactive_service_without_replay() {
        let root = temp_dir();
        std::fs::write(
            format!("{}/dashboard.json", root.path),
            br#"{"forgejo_url":"https://192.168.1.5"}"#,
        )
        .unwrap();
        std::fs::write(format!("{}/proxy.env", root.path), b"SODA_TLS=internal\n").unwrap();
        let observed = std::cell::RefCell::new(Vec::new());
        let run = FnRunner::new(|_, name: &str, args: &[String], input: Option<&[u8]>| {
            assert_eq!(name, "systemctl");
            assert_eq!(args.len(), 3);
            assert_eq!(args[0], "is-active");
            assert_eq!(args[1], "--quiet");
            assert!(input.is_none(), "configured guidance attempted a mutation");
            observed.borrow_mut().push(args[2].clone());
            if args[2] == "soda-dashboard.service" {
                return Err(Error::msg("synthetic inactive unit"));
            }
            Ok(Vec::new())
        });
        let (ctx, _flag) = Ctx::test();
        let err = configured_access(&ctx, &null_console(), &root.path, "missing-ca", &run).unwrap_err();
        assert!(err.to_string().contains("soda-dashboard.service"), "{err}");
        assert_eq!(observed.borrow().len(), 2);
    }

    #[test]
    fn setup_rollback_keeps_referenced_state() {
        let fail = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            Err(Error::msg("synthetic setup failure"))
        });
        let selected = SetupAddress { interface: "eth0".to_string(), address: "192.168.1.5".to_string() };
        let (ctx, _flag) = Ctx::test();
        // Referenced configuration preserved.
        let root = temp_dir();
        std::fs::write(format!("{}/dashboard.json", root.path), br#"{"operator":"kept"}"#).unwrap();
        std::fs::write(format!("{}/grant-key", root.path), b"preexisting-key").unwrap();
        let work = temp_dir();
        let err = execute_setup_and_activation(
            &ctx,
            &fail,
            &root.path,
            &work.path,
            "https://192.168.1.5",
            b"bootstrap-token",
            &selected,
        )
        .unwrap_err();
        assert!(err.to_string().contains("inspect the existing configuration"), "{err}");
        assert_eq!(std::fs::read(format!("{}/dashboard.json", root.path)).unwrap(), br#"{"operator":"kept"}"#);
        assert_eq!(std::fs::read(format!("{}/grant-key", root.path)).unwrap(), b"preexisting-key");
        // Orphan reservation cleaned.
        let root = temp_dir();
        std::fs::write(format!("{}/grant-key", root.path), b"orphan-key").unwrap();
        let work = temp_dir();
        let err = execute_setup_and_activation(
            &ctx,
            &fail,
            &root.path,
            &work.path,
            "https://192.168.1.5",
            b"bootstrap-token",
            &selected,
        )
        .unwrap_err();
        assert!(err.to_string().contains("rerunning configure is safe"), "{err}");
        for name in ["grant-key", "setup-started"] {
            assert!(
                matches!(
                    std::fs::symlink_metadata(format!("{}/{}", root.path, name)),
                    Err(err) if err.kind() == std::io::ErrorKind::NotFound
                ),
                "{name} survived rollback"
            );
        }
    }

    #[test]
    fn operator_token_validation() {
        assert!(valid_operator_token(b"synthetic-operator-token"));
        for bad in ["", " padded", "padded ", "a\nb", "a\rb", "a\0b", "\u{a0}nbsp\u{a0}"] {
            assert!(!valid_operator_token(bad.as_bytes()), "{bad:?}");
        }
    }

    fn read_until(
        master: &mut std::fs::File,
        transcript: &mut Vec<u8>,
        ctx: &Ctx,
        needle: &str,
    ) {
        use std::io::Read;
        use std::os::unix::io::AsRawFd;
        loop {
            if String::from_utf8_lossy(transcript).contains(needle) {
                return;
            }
            if ctx.err().is_some() {
                panic!("missing setup prompt: {needle}");
            }
            let mut fd = libc::pollfd { fd: master.as_raw_fd(), events: libc::POLLIN, revents: 0 };
            if unsafe { libc::poll(&mut fd, 1, 100) } > 0 && fd.revents & libc::POLLIN != 0 {
                let mut data = [0u8; 4096];
                if let Ok(n) = master.read(&mut data) {
                    transcript.extend_from_slice(&data[..n]);
                }
            }
        }
    }

    #[test]
    fn private_setup_keeps_credential_out_of_commands_and_transcript() {
        use std::io::Write;
        use std::os::unix::fs::PermissionsExt;
        for cancel_setup in [false, true] {
            let root = temp_dir();
            std::fs::write(format!("{}/installed", root.path), b"").unwrap();
            let pty = open_test_pty();
            let slave_path = pty.slave_path.clone();
            let mut master = pty.master;
            let (ctx, _flag) = Ctx::test();
            let ctx = ctx.with_timeout(std::time::Instant::now() + std::time::Duration::from_secs(15));
            const TOKEN: &str = "synthetic-operator-token";
            let mutations = std::sync::Arc::new(std::sync::atomic::AtomicUsize::new(0));
            let worker_mutations = mutations.clone();
            let root_path = root.path.clone();
            let worker_ctx = ctx.clone();
            let worker = std::thread::spawn(move || {
                let console = Console::open(&slave_path).unwrap();
                let run = FnRunner::new(|_, name: &str, args: &[String], input: Option<&[u8]>| {
                    assert!(input.is_none(), "credential reached unexpected stdin");
                    assert!(
                        !args.iter().any(|a| a.contains(TOKEN)),
                        "credential reached command arguments"
                    );
                    if name == "ip" {
                        return Ok(br#"[{"ifname":"eth0","flags":["UP"],"addr_info":[{"local":"192.168.1.5","scope":"global"}]}]"#.to_vec());
                    }
                    if name == "systemctl" {
                        assert_eq!(args.len(), 3, "setup guidance changed service state");
                        assert_eq!(args[0], "is-active");
                        assert_eq!(args[1], "--quiet");
                        return Ok(Vec::new());
                    }
                    worker_mutations.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                    if name == "/usr/bin/soda-setup" {
                        assert_eq!(args.len(), 6, "unexpected setup arguments: {args:?}");
                        assert_eq!(args[0], "--forgejo-url");
                        assert_eq!(args[1], "https://192.168.1.5");
                        assert_eq!(args[2], "--token-file");
                        assert_eq!(args[4], "--out");
                        let data = std::fs::read(&args[3]).unwrap();
                        let mode = std::fs::symlink_metadata(&args[3]).unwrap().permissions().mode() & 0o777;
                        assert_eq!(data, format!("{TOKEN}\n").as_bytes(), "token was not passed in a restricted file");
                        assert_eq!(mode, 0o600, "token was not passed in a restricted file");
                        let config = format!(r#"{{"forgejo_url":"{}"}}"#, args[1]);
                        std::fs::write(&args[5], config).unwrap();
                        return Ok(Vec::new());
                    }
                    assert_eq!(name, "/usr/bin/soda-activate", "unexpected activation: {name} {args:?}");
                    assert_eq!(args.join(" "), "--bind-ip 192.168.1.5 --local-tls");
                    std::fs::write(format!("{}/proxy.env", root_path), b"SODA_TLS=internal\n").unwrap();
                    Ok(Vec::new())
                });
                configure_private_install(
                    &worker_ctx,
                    &console,
                    &run,
                    &root_path,
                    &root_path,
                    &format!("{root_path}/not-yet-created-ca.crt"),
                )
            });
            let mut transcript = Vec::new();
            read_until(&mut master, &mut transcript, &ctx, "Address number, or cancel");
            master.write_all(b"1\n").unwrap();
            read_until(&mut master, &mut transcript, &ctx, "When Forgejo setup is complete");
            {
                let text = String::from_utf8_lossy(&transcript);
                assert!(text.contains("Required scope: read:user"), "bootstrap guidance requests unrelated token authority");
                assert!(!text.contains("write:admin"), "bootstrap guidance requests unrelated token authority");
                assert!(!text.contains("read:repository"), "bootstrap guidance requests unrelated token authority");
            }
            if cancel_setup {
                master.write_all(b"cancel\n").unwrap();
            } else {
                master.write_all(b"CONFIGURE SODA\n").unwrap();
                read_until(&mut master, &mut transcript, &ctx, "Operator Forgejo token: ");
                master.write_all(format!("{TOKEN}\n").as_bytes()).unwrap();
            }
            let err = worker.join().unwrap();
            if cancel_setup {
                assert!(err.is_err(), "cancelled setup succeeded");
                assert_eq!(mutations.load(std::sync::atomic::Ordering::SeqCst), 0, "cancelled setup made native changes");
                assert!(
                    matches!(
                        std::fs::symlink_metadata(format!("{}/setup-started", root.path)),
                        Err(e) if e.kind() == std::io::ErrorKind::NotFound
                    ),
                    "cancelled setup reserved an attempt"
                );
            } else {
                assert!(err.is_ok(), "setup failed: {err:?}, mutations {}", mutations.load(std::sync::atomic::Ordering::SeqCst));
                assert_eq!(mutations.load(std::sync::atomic::Ordering::SeqCst), 2);
                read_until(&mut master, &mut transcript, &ctx, "certificate is not available yet");
                let leftovers: Vec<_> = std::fs::read_dir(&root.path)
                    .unwrap()
                    .filter_map(|e| e.ok())
                    .filter(|e| e.file_name().to_string_lossy().starts_with("soda-setup-"))
                    .collect();
                assert!(leftovers.is_empty(), "operator token workdir persists: {leftovers:?}");
            }
            transcript.extend_from_slice(&drain_available(&mut master));
            assert!(
                !String::from_utf8_lossy(&transcript).contains(TOKEN),
                "operator token echoed to console"
            );
        }
    }
}
