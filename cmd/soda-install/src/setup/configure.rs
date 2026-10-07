use super::access::configured_access;
use super::address::{private_setup_origin, setup_addresses, SetupAddress};
use super::SBIN;

use crate::command::{failure_summary, Runner};
use crate::console::Console;
use crate::errors::{self, Error};
use crate::signal::Ctx;

fn check_preexisting_install(root: &str) -> Result<bool, Error> {
    if crate::execute::read_regular(&format!("{root}/installed"), 256).is_err() {
        return Err(Error::msg(
            "install the included Soda components before configuring browser access",
        ));
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
    let data = run
        .run(
            ctx,
            "ip",
            &[
                "-json".to_string(),
                "address".to_string(),
                "show".to_string(),
                "up".to_string(),
            ],
            None,
        )
        .map_err(|_| Error::msg("cannot inspect network addresses"))?;
    setup_addresses(&data)
}

fn prompt_setup_address(
    ctx: &Ctx,
    console: &Console,
    choices: &[SetupAddress],
) -> Result<SetupAddress, Error> {
    console.page("Private browser setup");
    console.print(
        "Use this SSH terminal to paste the Forgejo token when asked; input will be hidden.",
    );
    console.print("Select the appliance address your laptop can reach. No domain is needed.");
    for (i, choice) in choices.iter().enumerate() {
        console.print(format_args!(
            "{}. {} on {:?}",
            i + 1,
            choice.address,
            choice.interface
        ));
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
        console.print("Choose one of the listed address numbers.");
    }
}

pub(super) fn valid_operator_token(token: &[u8]) -> bool {
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
    console.print(format_args!("The final Soda address will be {origin}"));
    console.print(
        "Use a stable address or DHCP reservation. Changing it later needs explicit configuration maintenance.");
    console.print(format_args!(
        "If you have no SSH key access yet, cancel and run {} enroll-key at the local console.",
        crate::candidate::CANDIDATE_INSTALLER_BINARY
    ));
    console.print("From your laptop, connect with an SSH tunnel to the native Forgejo installer:");
    console.print(format_args!("ssh -L 33000:127.0.0.1:3000 root@{address}"));
    console.print(
        "Open http://localhost:33000 and complete Forgejo's own installation and administrator account setup.");
    console.print(
        "Keep its localhost browser URL for this bootstrap; activation below sets the final private URL.");
    console.print(
        "In Forgejo Settings > Applications, create the operator token described in the operator setup guide.");
    console.print("Required scope: read:user. No admin or repository scope is needed. Paste it here through the SSH terminal; input is hidden.");
    console.print(
        "Caddy will issue local HTTPS certificates. You will explicitly trust its public root certificate on your laptop.");
    let answer = console.ask(
        ctx,
        "When Forgejo setup is complete, type CONFIGURE SODA; anything else cancels",
    )?;
    if answer != "CONFIGURE SODA" {
        return Err(Error::msg(
            "browser setup cancelled; no Soda configuration written",
        ));
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
    let current = query_setup_addresses(ctx, run)
        .map_err(|_| Error::msg("cannot recheck the selected address"))?;
    if current.iter().any(|choice| choice == selected) {
        return Ok(());
    }
    Err(Error::msg(
        "selected network address changed; restart browser setup",
    ))
}

fn make_setup_workdir(temporary: &str) -> Result<tempfile::TempDir, Error> {
    use std::os::unix::fs::PermissionsExt;
    tempfile::Builder::new()
        .prefix("soda-setup-")
        .permissions(std::fs::Permissions::from_mode(0o700))
        .tempdir_in(temporary)
        .map_err(|err| errors::path_error("mkdir", temporary, err))
}

pub(super) fn execute_setup_and_activation(
    ctx: &Ctx,
    run: &dyn Runner,
    root: &str,
    temporary: &str,
    origin: &str,
    token: &[u8],
    selected: &SetupAddress,
) -> Result<(), Error> {
    let work = make_setup_workdir(temporary)?;
    let result = (|| -> Result<(), Error> {
        let work = work.path().to_string_lossy();
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
            &[
                "--bind-ip".to_string(),
                selected.address.clone(),
                "--local-tls".to_string(),
            ],
            None,
        ) {
            return Err(Error::msg(format!(
                "private activation failed; preserve the existing configuration for inspection. {}",
                failure_summary(&err)
            )));
        }
        Ok(())
    })();
    let work_path = work.path().to_string_lossy().into_owned();
    let cleanup = work
        .close()
        .map_err(|err| errors::path_error("remove", &work_path, err));
    match (result, cleanup) {
        (Ok(()), Ok(())) => Ok(()),
        (Err(primary), Ok(())) => Err(primary),
        (Ok(()), Err(cleanup)) => Err(cleanup),
        (Err(primary), Err(cleanup)) => Err(Error::msg(format!(
            "{primary}; temporary cleanup failed: {cleanup}"
        ))),
    }
}

pub(super) fn configure_private_install(
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
    let write_err = file
        .write(data)
        .err()
        .map(|e| errors::path_error("write", path, e));
    use std::os::unix::io::IntoRawFd;
    let close_err = if unsafe { libc::close(file.into_raw_fd()) } != 0 {
        let errno = unsafe { *libc::__errno_location() };
        Some(errors::path_error(
            "close",
            path,
            std::io::Error::from_raw_os_error(errno),
        ))
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
