use super::*;
use crate::command::FnRunner;

fn device(name: &str) -> BlockDevice {
    BlockDevice {
        name: name.to_string(),
        kname: name.to_string(),
        device_type: "disk".to_string(),
        tran: "nvme".to_string(),
        size: 64 << 30,
        model: "SODA".to_string(),
        serial: "123".to_string(),
        wwn: "wwn-1".to_string(),
        major_minor: "259:0".to_string(),
        read_only: false,
        mountpoints: Some(vec![None]),
        fstype: String::new(),
        uuid: String::new(),
        partuuid: String::new(),
        children: Vec::new(),
    }
}

#[test]
fn unused_taxonomy() {
    assert_eq!(unused(&device("/dev/nvme0n1")), "");
    let mut d = device("/dev/nvme0n1");
    d.mountpoints = None;
    assert_eq!(unused(&d), "incomplete or unsupported device inventory");
    let d = device("nvme0n1");
    assert_eq!(unused(&d), "incomplete or unsupported device inventory");
    let mut d = device("/dev/nvme0n1");
    d.size = 0;
    assert_eq!(unused(&d), "incomplete or unsupported device inventory");
    let mut d = device("/dev/nvme0n1");
    d.read_only = true;
    assert_eq!(unused(&d), "read-only device");
    let mut d = device("/dev/nvme0n1");
    d.mountpoints = Some(vec![Some("/".to_string())]);
    assert_eq!(unused(&d), "mounted filesystem or active swap");
    let mut d = device("/dev/nvme0n1");
    d.fstype = "iso9660".to_string();
    assert_eq!(unused(&d), "installation/optical media");
    let mut d = device("/dev/nvme0n1");
    d.fstype = "crypto_LUKS".to_string();
    assert_eq!(
        unused(&d),
        "unrecognized, multi-device or encrypted storage requires separate operator handling"
    );
    let mut d = device("/dev/nvme0n1");
    let mut child = device("/dev/nvme0n1p1");
    child.device_type = "part".to_string();
    child.mountpoints = Some(vec![Some("/boot".to_string())]);
    d.children = vec![child];
    assert_eq!(unused(&d), "mounted filesystem or active swap");
    let mut d = device("/dev/nvme0n1");
    let mut child = device("/dev/mapper/x");
    child.device_type = "lvm".to_string();
    d.children = vec![child];
    assert_eq!(unused(&d), "active mapped/stacked device");
}

#[test]
fn live_media_parsing() {
    // Ported from the Go mountinfo/cmdline fixtures.
    let mountinfo = b"22 1 0:20 / /run/initramfs/live rw - iso9660 /dev/sr0 ro\n30 1 8:1 / /boot rw - ext4 /dev/sda1 rw\n";
    let media = parse_live_media_disks(mountinfo, b"", &|dev| {
        assert_eq!(dev, "/dev/sr0");
        "/dev/sr0".to_string()
    });
    assert!(media.contains("/dev/sr0"));
    let media = parse_live_media_disks(
        b"garbage\nno-separator-here a b c d",
        b"root=live:/dev/sdb",
        &|dev| {
            assert_eq!(dev, "/dev/sdb");
            "/dev/sda".to_string()
        },
    );
    assert!(media.contains("/dev/sda"));
    let media = parse_live_media_disks(b"", b"root=/dev/sda1 quiet", &|_| panic!("no mapping"));
    assert!(media.is_empty());
}

#[test]
fn parent_disk_sysfs_walk() {
    let root = std::env::temp_dir().join(format!("soda-install-sys-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&root);
    std::fs::create_dir_all(&root).unwrap();
    // Whole disk: link whose directory base is `block`.
    std::os::unix::fs::symlink("../../devices/pci/block/sda", root.join("sda")).unwrap();
    // Partition: link whose directory base names the disk.
    std::os::unix::fs::symlink("../../devices/pci/block/sda/sda1", root.join("sda1")).unwrap();
    assert_eq!(
        parent_disk_of_sys(root.to_str().unwrap(), "/dev/sda1"),
        "/dev/sda"
    );
    assert_eq!(
        parent_disk_of_sys(root.to_str().unwrap(), "/dev/sda"),
        "/dev/sda"
    );
    assert_eq!(
        parent_disk_of_sys(root.to_str().unwrap(), "/dev/missing"),
        ""
    );
    let _ = std::fs::remove_dir_all(&root);
}

#[test]
fn summary_and_same_disk() {
    let d = device("/dev/nvme0n1");
    assert_eq!(
        disk_summary(&d),
        "\"/dev/nvme0n1\" | 64.0 GiB | model \"SODA\" | serial \"123\" | WWN \"wwn-1\""
    );
    let disk = Disk {
        device: d.clone(),
        sequence: "7".to_string(),
        blocked: String::new(),
        removable: false,
    };
    assert!(same_disk(&disk, std::slice::from_ref(&disk)).is_ok());
    let mut changed = disk.clone();
    changed.sequence = "8".to_string();
    assert!(same_disk(&disk, &[changed]).is_err());
    let mut unsequenced = disk.clone();
    unsequenced.sequence.clear();
    assert!(same_disk(&unsequenced, std::slice::from_ref(&disk)).is_err());
    assert_eq!(
        same_disk(&disk, &[]).unwrap_err().to_string(),
        "selected disk disappeared; no installation started"
    );
}

#[test]
fn scan_decodes_and_blocks() {
    let lsblk = r#"{"blockdevices":[
{"name":"/dev/nvme0n1","kname":"/dev/nvme0n1","type":"disk","tran":"nvme","size":68719476736,"model":"SODA","serial":"1","wwn":"w","maj:min":"259:0","ro":false,"mountpoints":[null],"fstype":"","uuid":"","partuuid":"","children":[]},
{"name":"/dev/sr0","kname":"/dev/sr0","type":"rom","tran":"sata","size":0,"model":"","serial":"","wwn":"","maj:min":"11:0","ro":true,"mountpoints":null,"fstype":"iso9660","uuid":"","partuuid":""}
]}"#;
    let root = std::env::temp_dir().join(format!("soda-install-scan-{}", std::process::id()));
    let _ = std::fs::remove_dir_all(&root);
    std::fs::create_dir_all(root.join("nvme0n1/holders")).unwrap();
    std::fs::write(root.join("nvme0n1/diskseq"), b"42\n").unwrap();
    let (ctx, _flag) = Ctx::test();
    let runner = FnRunner::new(|_, name, args, _| {
        assert_eq!(name, "lsblk");
        assert_eq!(args[0], "--json");
        Ok(lsblk.as_bytes().to_vec())
    });
    let media = std::collections::BTreeSet::new();
    let disks = scan_disks_at(&ctx, &runner, root.to_str().unwrap(), &media).unwrap();
    assert_eq!(disks.len(), 1);
    assert_eq!(disks[0].sequence, "42");
    assert!(disks[0].blocked.is_empty());
    assert!(!disks[0].removable);

    let mut media = std::collections::BTreeSet::new();
    media.insert("/dev/nvme0n1".to_string());
    let disks = scan_disks_at(&ctx, &runner, root.to_str().unwrap(), &media).unwrap();
    assert_eq!(disks[0].blocked, "current installer media");

    let runner = FnRunner::new(|_, _, _, _| {
        Err(Error::CmdExit {
            name: "lsblk".to_string(),
            code: 1,
            interrupted: false,
        })
    });
    assert_eq!(
        scan_disks_at(
            &ctx,
            &runner,
            root.to_str().unwrap(),
            &std::collections::BTreeSet::new()
        )
        .unwrap_err()
        .to_string(),
        "cannot inventory block devices"
    );
    let _ = std::fs::remove_dir_all(&root);
}
