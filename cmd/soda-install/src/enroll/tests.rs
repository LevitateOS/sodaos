use super::native_config::enrollment_binary;
use super::test_support::TEST_KEY;
use super::*;

#[test]
fn enrollment_input_matrix() {
    for value in [
        TEST_KEY.to_string(),
        format!("{TEST_KEY}\n"),
        format!("{TEST_KEY} laptop\n"),
    ] {
        assert_eq!(
            enrollment_public_key(value.as_bytes()).unwrap(),
            TEST_KEY,
            "{value:?}"
        );
    }
    let big = "x".repeat(ENROLLMENT_KEY_LIMIT + 1);
    for value in [
        String::new(),
        format!("{TEST_KEY}\n\n"),
        format!("{TEST_KEY}\n{TEST_KEY}\n"),
        format!("command=\"sh\" {TEST_KEY}"),
        "-----BEGIN OPENSSH PRIVATE KEY-----".to_string(),
        big,
        format!("{TEST_KEY}\r\n"),
        format!("{TEST_KEY}\x00"),
    ] {
        assert!(
            enrollment_public_key(value.as_bytes()).is_err(),
            "{value:?}"
        );
    }
}

#[test]
fn enrollment_private_address_and_client() {
    for address in ["10.0.0.8", "172.16.3.4", "192.168.1.20"] {
        let command = enrollment_client_command(address).unwrap();
        assert!(
            command.contains(&format!("root@{address} < ~/.ssh/id_ed25519.pub")),
            "{command}"
        );
        assert!(command.contains("ControlPath=none"), "{command}");
        assert!(command.contains("ClearAllForwardings=yes"), "{command}");
    }
    for address in [
        "",
        "0.0.0.0",
        "127.0.0.1",
        "169.254.1.1",
        "8.8.8.8",
        "100.64.0.1",
        "::1",
        "fd00::1",
        "192.168.1.1\nPermitTTY yes",
        "192.168.1.01",
    ] {
        assert!(enrollment_client_command(address).is_err(), "{address:?}");
    }
}

#[test]
fn enrollment_native_policy() {
    let config = enrollment_config();
    for line in [
        "PermitRootLogin yes",
        "AuthenticationMethods password",
        "PasswordAuthentication yes",
        "UsePAM no",
        "KbdInteractiveAuthentication no",
        "PubkeyAuthentication no",
        "PermitEmptyPasswords no",
        "DisableForwarding yes",
        "PermitTunnel no",
        "PermitTTY no",
        "PermitUserRC no",
        "PermitUserEnvironment no",
        "MaxSessions 1",
        "MaxAuthTries 3",
        "AuthorizedKeysFile none",
        &format!("ForceCommand {} enrollment-receive", enrollment_binary()),
    ] {
        assert!(config.contains(&format!("\n{line}\n")), "missing {line}");
    }
    for forbidden in [
        "Include ",
        "AcceptEnv ",
        "Subsystem ",
        "ListenAddress ",
        "Match ",
        "SetEnv ",
    ] {
        assert!(!config.contains(forbidden), "unexpected {forbidden:?}");
    }
    let args = enrollment_start_args().join(" ");
    for required in [
        "--unit=soda-key-enrollment.service",
        "--property=RuntimeMaxSec=300s",
        "--property=KillMode=control-group",
        "--property=SendSIGKILL=yes",
        "--property=Restart=no",
        "--property=TimeoutStopSec=2s",
    ] {
        assert!(args.contains(required), "missing {required}");
    }
}

#[test]
fn enrollment_native_socket_ownership() {
    let config = enrollment_socket_unit_config("192.168.1.20").unwrap();
    for line in [
        "BindsTo=soda-key-enrollment.service",
        "After=soda-key-enrollment.service",
        "ListenStream=192.168.1.20:22222",
        "Accept=yes",
        "MaxConnections=2",
    ] {
        assert!(config.contains(&format!("\n{line}\n")), "missing {line}");
    }
    assert!(enrollment_socket_unit_config("0.0.0.0\nAccept=no").is_err());
    let connection = enrollment_template_unit_config();
    for line in [
        "BindsTo=soda-key-enrollment.service soda-key-enrollment.socket",
        "After=soda-key-enrollment.service soda-key-enrollment.socket",
        "StandardInput=socket",
        "StandardOutput=socket",
        "Slice=system.slice",
        "RuntimeMaxSec=300s",
        "KillMode=control-group",
        "TimeoutStopSec=2s",
        "SendSIGKILL=yes",
        "Restart=no",
        "ExecStart=/usr/sbin/sshd -i -e -f /run/soda-key-enrollment/sshd_config",
    ] {
        assert!(
            connection.contains(&format!("\n{line}\n")),
            "missing {line}"
        );
    }
    assert!(!(config + &connection).contains("[Install]"));
}

#[test]
fn enrollment_peer_service_provenance() {
    let unit = enrollment_peer_unit(b"0::/system.slice/soda-key-enrollment.service\n").unwrap();
    assert_eq!(unit, b"soda-key-enrollment.service");
    assert!(!enrollment_receiver_unit(&unit));
    let unit = enrollment_peer_unit(
        b"0::/system.slice/soda-key-enrollment@0-192.168.1.20:22222-192.168.1.30:43123.service\n",
    )
    .unwrap();
    assert!(enrollment_receiver_unit(&unit));
    for group in [
        "0::/user.slice/user-0.slice/session-1.scope\n",
        "0::/system.slice/soda-key-enrollment@x.service/subgroup\n",
        "0::/system.slice/soda-key-enrollment@x.service\n0::/system.slice/sshd.service\n",
    ] {
        assert!(enrollment_peer_unit(group.as_bytes()).is_err(), "{group:?}");
    }
    for unit in [
        "soda-key-enrollment@.service",
        "sshd.service",
        "soda-key-enrollment-other@x.service",
        "soda-key-enrollment@x.service/child",
        "soda-key-enrollment@x.service\n",
    ] {
        assert!(!enrollment_receiver_unit(unit.as_bytes()), "{unit:?}");
    }
}
