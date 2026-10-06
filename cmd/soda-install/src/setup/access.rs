use super::address::private_setup_origin;
use super::local_ca_fingerprint;

use crate::command::Runner;
use crate::console::Console;
use crate::errors::Error;
use crate::fmtx::Arg;
use crate::jsongo::{parse, Soft};
use crate::signal::Ctx;

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
    let config =
        Soft::new(&value).map_err(|_| Error::msg("invalid installed browser configuration"))?;
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
    console.print(
        "Local CA certificate SHA-256: %s",
        &[Arg::Str(&fingerprint)],
    );
    console.print(
        "Copy only the public root.crt file over your verified SSH connection:",
        &[],
    );
    let host = String::from_utf8_lossy(&origin.host);
    console.print(
        "scp root@%s:%s ./soda-local-ca.crt",
        &[Arg::Str(&host), Arg::Str(ca_path)],
    );
    console.print("Compare its certificate fingerprint, then trust it in your laptop/browser certificate settings. Never copy the CA private key.", &[]);
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
        let host = origin.hostname();
        let hostname = String::from_utf8_lossy(&host);
        let expected = private_setup_origin(&hostname).unwrap_or_default();
        if address.strip_suffix('/').unwrap_or(&address) != expected {
            return Err(Error::msg(
                "local TLS must use the selected private IP origin",
            ));
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
