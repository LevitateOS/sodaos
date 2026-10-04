//! SELinux labeling for the temporary enrollment listener and its
//! sshd configuration. Non-enforcing systems skip silently.

// enrollmentSELinuxPortType labels the temporary import listener as what it
// is: a genuine OpenSSH service. The bind runs in a context that may use
// ssh_port_t, while the default unreserved_port_t bind is denied.
const ENROLLMENT_SELINUX_PORT_TYPE: &str = "ssh_port_t";

// enrollmentPortLabeled reports whether a semanage port listing already
// assigns the enrollment port to the SSH port type.
fn enrollment_port_labeled(listing: &[u8]) -> bool {
    for line in String::from_utf8_lossy(listing).split('\n') {
        let fields: Vec<&str> = line.split_whitespace().collect();
        if fields.len() < 3 || fields[0] != ENROLLMENT_SELINUX_PORT_TYPE || fields[1] != "tcp" {
            continue;
        }
        for port in fields[2..].join(" ").split(',') {
            if port.trim() == super::ENROLLMENT_PORT {
                return true;
            }
        }
    }
    false
}

// ensureEnrollmentPortLabel makes the enrollment bind survivable where
// SELinux enforces. Non-enforcing systems skip silently: labeling is
// meaningless without enforcement, and semanage may not exist there.
pub fn ensure_enrollment_port_label(
    ctx: &crate::signal::Ctx,
    run: &dyn crate::command::Runner,
    enforcing: bool,
) -> Result<(), crate::errors::Error> {
    use crate::errors::Error;
    if !enforcing {
        return Ok(());
    }
    let listing = match run.run(ctx, "semanage", &["port".to_string(), "-l".to_string()], None) {
        Ok(listing) => listing,
        Err(_) => {
            return Err(Error::msg(format!(
                "SELinux is enforcing but the SSH port labeling cannot be inspected; install policycoreutils-python-utils or label tcp/{} {} manually",
                super::ENROLLMENT_PORT,
                ENROLLMENT_SELINUX_PORT_TYPE
            )))
        }
    };
    if enrollment_port_labeled(&listing) {
        return Ok(());
    }
    if let Err(err) = run.run(
        ctx,
        "semanage",
        &[
            "port".to_string(),
            "-a".to_string(),
            "-t".to_string(),
            ENROLLMENT_SELINUX_PORT_TYPE.to_string(),
            "-p".to_string(),
            "tcp".to_string(),
            super::ENROLLMENT_PORT.to_string(),
        ],
        None,
    ) {
        return Err(Error::msg(format!(
            "label tcp/{} {}: {err}",
            super::ENROLLMENT_PORT,
            ENROLLMENT_SELINUX_PORT_TYPE
        )));
    }
    Ok(())
}

// labelEnrollmentConfig lets the per-connection sshd read its config. The
// state directory carries var_run_t, which sshd cannot read; mirror the
// native /etc/ssh/sshd_config type instead.
pub fn label_enrollment_config(
    ctx: &crate::signal::Ctx,
    run: &dyn crate::command::Runner,
    enforcing: bool,
) -> Result<(), crate::errors::Error> {
    use crate::errors::Error;
    if let Err(err) = run.run(
        ctx,
        "chcon",
        &["-t".to_string(), "etc_t".to_string(), super::ENROLLMENT_CONFIG_PATH.to_string()],
        None,
    ) {
        if !enforcing {
            return Ok(());
        }
        return Err(Error::msg(format!("label enrollment SSH config: {err}")));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::command::FnRunner;
    use crate::errors::Error;
    use crate::signal::Ctx;

    #[test]
    fn port_labeled_matrix() {
        for (listing, want) in [
            ("ssh_port_t tcp 22\n", false),
            ("ssh_port_t tcp 22222, 22\n", true),
            ("ssh_port_t                     tcp      22222\n", true),
            ("unreserved_port_t tcp 22222\n", false),
            ("ssh_port_t udp 22222\n", false),
            ("ssh_port_t tcp 2222\n", false),
            ("", false),
        ] {
            assert_eq!(enrollment_port_labeled(listing.as_bytes()), want, "{listing:?}");
        }
    }

    #[test]
    fn ensure_port_label_flows() {
        let (ctx, _flag) = Ctx::test();
        // Permissive skips without running semanage.
        let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            panic!("semanage must not run without enforcement")
        });
        ensure_enrollment_port_label(&ctx, &run, false).unwrap();
        // Labeled port needs no change: list only.
        let calls = std::cell::RefCell::new(Vec::new());
        let run = FnRunner::new(|_, name: &str, args: &[String], _| {
            calls.borrow_mut().push(format!("{name} {}", args.join(" ")));
            Ok(b"ssh_port_t tcp 22222, 22\n".to_vec())
        });
        ensure_enrollment_port_label(&ctx, &run, true).unwrap();
        assert_eq!(calls.borrow().len(), 1);
        // Missing label is added.
        let calls = std::cell::RefCell::new(Vec::new());
        let run = FnRunner::new(|_, name: &str, args: &[String], _| {
            calls.borrow_mut().push(format!("{name} {}", args.join(" ")));
            if !args.is_empty() && args[0] == "port" && args.get(1).map(String::as_str) != Some("-a") {
                return Ok(b"ssh_port_t tcp 22\n".to_vec());
            }
            Ok(Vec::new())
        });
        ensure_enrollment_port_label(&ctx, &run, true).unwrap();
        assert_eq!(calls.borrow().len(), 2);
        assert!(calls.borrow()[1].contains("port -a -t ssh_port_t -p tcp 22222"), "{:?}", calls.borrow());
        // Missing semanage stays loud when enforcing.
        let run = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            Err(Error::CmdExit { name: "semanage".to_string(), code: 127, interrupted: false })
        });
        assert!(ensure_enrollment_port_label(&ctx, &run, true).is_err());
    }

    #[test]
    fn label_config_flows() {
        let (ctx, _flag) = Ctx::test();
        let run = FnRunner::new(|_, name: &str, args: &[String], _| {
            assert_eq!(name, "chcon");
            assert_eq!(args, &["-t".to_string(), "etc_t".to_string(), super::super::ENROLLMENT_CONFIG_PATH.to_string()]);
            Ok(Vec::new())
        });
        label_enrollment_config(&ctx, &run, true).unwrap();
        let failing = FnRunner::new(|_, _, _, _| -> Result<Vec<u8>, Error> {
            Err(Error::CmdExit { name: "chcon".to_string(), code: 1, interrupted: false })
        });
        assert!(label_enrollment_config(&ctx, &failing, true).is_err());
        label_enrollment_config(&ctx, &failing, false).unwrap();
    }
}
