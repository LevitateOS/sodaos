use super::address::private_setup_origin;
use super::local_ca::local_ca_fingerprint;

use crate::command::Runner;
use crate::console::Console;
use crate::errors::Error;
use crate::signal::Ctx;
use url::Url;

fn valid_installed_https_origin(origin: &Url, raw: &str) -> bool {
    if origin.scheme() != "https"
        || origin.host().is_none()
        || origin.username() != ""
        || origin.password().is_some()
        || origin.query().is_some_and(|query| !query.is_empty())
        || origin
            .fragment()
            .is_some_and(|fragment| !fragment.is_empty())
    {
        return false;
    }
    let Some((_, rest)) = raw.split_once("://") else {
        return false;
    };
    let authority = rest.split(['/', '?', '#']).next().unwrap_or("");
    if authority.is_empty()
        || authority.contains('@')
        || authority.starts_with('[')
            && authority
                .split_once(']')
                .is_some_and(|(inside, _)| inside.contains('%'))
    {
        return false;
    }
    let raw_path = rest
        .find('/')
        .map(|at| rest[at..].split(['?', '#']).next().unwrap_or(""))
        .unwrap_or("");
    if !matches!(raw_path, "" | "/") || !matches!(origin.path(), "" | "/") {
        return false;
    }
    !raw.bytes().any(|b| b <= 0x20 || b == 0x7f || b == b'\\') && valid_percent_escapes(raw)
}

fn valid_percent_escapes(value: &str) -> bool {
    let bytes = value.as_bytes();
    let mut at = 0;
    while at < bytes.len() {
        if bytes[at] == b'%' {
            if at + 2 >= bytes.len()
                || !bytes[at + 1].is_ascii_hexdigit()
                || !bytes[at + 2].is_ascii_hexdigit()
            {
                return false;
            }
            at += 3;
        } else {
            at += 1;
        }
    }
    true
}

fn installed_forgejo_origin(root: &str) -> Result<(Url, String), Error> {
    let data = crate::execute::read_regular(&format!("{root}/dashboard.json"), 65536)
        .map_err(|_| Error::msg("cannot read installed browser address"))?;
    let text = String::from_utf8_lossy(&data);
    let config: serde_json::Value = serde_json::from_str(&text)
        .map_err(|_| Error::msg("invalid installed browser configuration"))?;
    let config = config
        .as_object()
        .ok_or_else(|| Error::msg("invalid installed browser configuration"))?;
    let url = match config.get("forgejo_url") {
        None | Some(serde_json::Value::Null) => String::new(),
        Some(serde_json::Value::String(value)) => value.clone(),
        Some(_) => return Err(Error::msg("invalid installed browser configuration")),
    };
    let origin = Url::parse(&url).map_err(|_| Error::msg("invalid installed HTTPS address"))?;
    if !valid_installed_https_origin(&origin, &url) {
        return Err(Error::msg("invalid installed HTTPS address"));
    }
    Ok((origin, url))
}

fn uses_internal_tls(root: &str) -> Result<bool, Error> {
    let proxy = crate::execute::read_regular(&format!("{root}/proxy.env"), 65536)
        .map_err(|_| Error::msg("cannot inspect the installed TLS mode"))?;
    Ok(proxy
        .split(|b| *b == b'\n')
        .any(|line| line == b"SODA_TLS=internal"))
}

fn confirm_active_browser_units(ctx: &Ctx, run: &dyn Runner) -> Result<(), Error> {
    for unit in [
        "forgejo.service",
        "soda-dashboard.service",
        "soda-proxy.service",
    ] {
        if run
            .run(
                ctx,
                "systemctl",
                &[
                    "is-active".to_string(),
                    "--quiet".to_string(),
                    unit.to_string(),
                ],
                None,
            )
            .is_err()
        {
            return Err(Error::msg(format!("private activation was requested, but {unit} is not confirmed active; inspect its native service state, then rerun configure for read-only guidance; setup was not replayed")));
        }
    }
    Ok(())
}

fn print_local_ca_guidance(console: &Console, address: &str, ca_path: &str) -> Result<(), Error> {
    let certificate = match crate::execute::read_regular(ca_path, 16384) {
        Ok(certificate) => certificate,
        Err(_) => {
            // Active units without trust material are not a ready browser
            // address: say so before any Open guidance, not after it.
            console.print("The browser services report active, but the local CA certificate is not available yet; do not open the browser address until the trust material below exists.");
            console.print("Inspect soda-proxy.service, then run configure again to show the trust instructions. Existing setup will not be replayed.");
            return Ok(());
        }
    };
    let fingerprint = local_ca_fingerprint(&certificate)?;
    console.print(format_args!("The browser services report active. Open {address:?} only after completing the client trust below; browser login still needs verification."));
    console.print(
        format_args!("Local CA certificate SHA-256: {fingerprint}"),
    );
    console.print(
        "Copy only the public root.crt file over your verified SSH connection:");
    let host = address
        .split_once("://")
        .map(|(_, rest)| rest.split(['/', '?', '#']).next().unwrap_or(""))
        .unwrap_or("");
    console.print(
        format_args!("scp root@{host:?}:{ca_path:?} ./soda-local-ca.crt"),
    );
    console.print("Compare its certificate fingerprint, then trust it in your laptop/browser certificate settings. Never copy the CA private key.");
    Ok(())
}

pub(super) fn configured_access(
    ctx: &Ctx,
    console: &Console,
    root: &str,
    ca_path: &str,
    run: &dyn Runner,
) -> Result<(), Error> {
    let (origin, address) = installed_forgejo_origin(root)?;
    let local = uses_internal_tls(root)?;
    if local {
        let hostname = origin
            .host()
            .map(|host| host.to_string())
            .unwrap_or_default();
        let expected = private_setup_origin(&hostname).unwrap_or_default();
        if address.strip_suffix('/').unwrap_or(&address) != expected {
            return Err(Error::msg(
                "local TLS must use the selected private IP origin",
            ));
        }
    }
    confirm_active_browser_units(ctx, run)?;
    if local {
        print_local_ca_guidance(console, &address, ca_path)?;
    } else {
        console.print(format_args!("The browser services report active. Open {address:?} after setting up client trust for your supplied certificate; browser login still needs verification."));
    }
    console.print("Sign in to native Forgejo, create or choose a repository, and open Sodaspaces to create and join its development environment.");
    console.print("Verify a browser terminal in that project. Opening this setup screen is not a completed project/access test.");
    Ok(())
}
