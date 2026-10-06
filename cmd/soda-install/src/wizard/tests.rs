use super::*;
use crate::console::test_support::*;
use std::io::Write;

#[test]
fn selections_and_phrases() {
    let disk = Disk {
        device: crate::disks::BlockDevice {
            name: "/dev/sda".to_string(),
            kname: "/dev/sda".to_string(),
            device_type: "disk".to_string(),
            tran: String::new(),
            size: 0,
            model: String::new(),
            serial: String::new(),
            wwn: String::new(),
            major_minor: "8:0".to_string(),
            read_only: false,
            mountpoints: Some(vec![]),
            fstype: String::new(),
            uuid: String::new(),
            partuuid: String::new(),
            children: Vec::new(),
        },
        sequence: "1".to_string(),
        blocked: String::new(),
        removable: false,
    };
    assert!(select_disk(std::slice::from_ref(&disk), "1").is_ok());
    assert_eq!(
        select_disk(std::slice::from_ref(&disk), "0").unwrap_err(),
        "Choose an available disk number."
    );
    assert_eq!(
        select_disk(std::slice::from_ref(&disk), "2").unwrap_err(),
        "Choose an available disk number."
    );
    assert_eq!(
        select_disk(std::slice::from_ref(&disk), "x").unwrap_err(),
        "Choose an available disk number."
    );
    assert_eq!(
        select_disk(std::slice::from_ref(&disk), "+1")
            .unwrap()
            .device
            .name,
        "/dev/sda"
    );
    let mut blocked = disk.clone();
    blocked.blocked = "busy".to_string();
    assert_eq!(
        select_disk(&[blocked], "1").unwrap_err(),
        "Disk 1 is unavailable: busy. Choose an available disk number."
    );
    assert_eq!(erase_phrase(&disk), "ERASE /dev/sda");
    let mut removable = disk.clone();
    removable.removable = true;
    assert_eq!(erase_phrase(&removable), "ERASE REMOVABLE /dev/sda");
    assert!(valid_password(b"twelve-chars!!", b"twelve-chars!!"));
    assert!(!valid_password(b"short", b"short"));
    assert!(!valid_password(b"twelve-chars!!", b"twelve-chars!?"));
    assert!(!valid_password(b"\xff\xfe-twelve!!", b"\xff\xfe-twelve!!"));
    assert_eq!(password_feedback(b"a", b"b"), "Passwords do not match.");
    assert_eq!(
        password_feedback(b"short", b"short"),
        "Use at least 12 characters."
    );
    assert_eq!(password_feedback(b"twelve-chars!!", b"twelve-chars!!"), "");
    assert_eq!(
        password_feedback(b"\xff\xfe-twelve!!", b"\xff\xfe-twelve!!"),
        "Use at least 12 characters."
    );
}

#[test]
fn full_wizard_flow() {
    let pty = open_pty();
    let slave = pty.slave_path.clone();
    let mut master = pty.master;
    let worker = std::thread::spawn(move || {
        let console = Console::open(&slave).unwrap();
        let (ctx, _flag) = Ctx::test();
        let runner = crate::command::FnRunner::new(|_, name, args, _| {
            if name == "ip" {
                return Ok(b"lo UP 127.0.0.1/8\n".to_vec());
            }
            if name == "openssl" {
                assert_eq!(
                    args,
                    &["passwd".to_string(), "-6".to_string(), "-stdin".to_string()]
                );
                return Ok(b"$6$salt$hash\n".to_vec());
            }
            panic!("unexpected command {name}");
        });
        let disk = Disk {
            device: crate::disks::BlockDevice {
                name: "/dev/sda".to_string(),
                kname: "/dev/sda".to_string(),
                device_type: "disk".to_string(),
                tran: String::new(),
                size: 64 << 30,
                model: "M".to_string(),
                serial: "S".to_string(),
                wwn: "W".to_string(),
                major_minor: "8:0".to_string(),
                read_only: false,
                mountpoints: Some(vec![None]),
                fstype: String::new(),
                uuid: String::new(),
                partuuid: String::new(),
                children: Vec::new(),
            },
            sequence: "9".to_string(),
            blocked: String::new(),
            removable: false,
        };
        collect_disk_install_choices(
            &ctx,
            &console,
            &runner,
            &|_, _| Ok(vec![disk.clone()]),
            10 << 20,
        )
    });
    let mut output = read_until(
        &mut master,
        b"Type keep, edit, back, restart, or cancel: ",
        3000,
    );
    master.write_all(b"keep\n").unwrap();
    output.extend(read_until(
        &mut master,
        b"Type yes to use them, edit, back, restart, or cancel: ",
        3000,
    ));
    master.write_all(b"yes\n").unwrap();
    output.extend(read_until(
        &mut master,
        b"Disk number, back, restart, or cancel: ",
        3000,
    ));
    master.write_all(b"1\n").unwrap();
    output.extend(read_until(
        &mut master,
        b"Hostname [soda], back, restart, or cancel: ",
        3000,
    ));
    master.write_all(b"\n").unwrap();
    output.extend(read_until(&mut master, b"Password: ", 3000));
    master.write_all(b"twelve-chars!!\n").unwrap();
    output.extend(read_until(&mut master, b"Confirm password: ", 3000));
    master.write_all(b"twelve-chars!!\n").unwrap();
    output.extend(read_until(
        &mut master,
        b"Project IPv4 subnet [10.89.0.0/24], back, restart, or cancel: ",
        3000,
    ));
    master.write_all(b"\n").unwrap();
    output.extend(read_until(
        &mut master,
        b"Type exactly ERASE /dev/sda, back, restart, or cancel: ",
        3000,
    ));
    master.write_all(b"ERASE /dev/sda\n").unwrap();
    let choices = worker.join().unwrap().unwrap();
    output.extend(drain_idle(&mut master, 200));
    assert_eq!(choices.hostname, "soda");
    assert_eq!(choices.subnet, "10.89.0.0/24");
    assert_eq!(choices.password_hash, "$6$salt$hash");
    assert!(!choices.removable_ok);
    let text = String::from_utf8_lossy(&output);
    assert!(text.contains("Final review"));
    assert!(text.contains("ERASE ALL DATA on:"));
    assert!(text.contains("10.0 MiB verified"));
}
