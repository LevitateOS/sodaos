//! Block-device inventory: `lsblk` decoding, use/refusal checks, kernel
//! identity (`diskseq`, holders), live-media exclusion, and summaries.

use soda_json::JsonValue;

use crate::command::Runner;
use crate::errors::Error;
use crate::fmtx::{fold_eq_ascii, go_fields, go_trim_space, sprintf, Arg};
use crate::jsongo::Soft;
use crate::pathx;
use crate::signal::Ctx;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct BlockDevice {
    pub name: String,
    pub kname: String,
    pub device_type: String,
    pub tran: String,
    pub size: u64,
    pub model: String,
    pub serial: String,
    pub wwn: String,
    pub major_minor: String,
    pub read_only: bool,
    /// `None` mirrors a null/missing JSON array; `Some` preserves even an
    /// empty one, because `sameDisk` compares inventories deeply.
    pub mountpoints: Option<Vec<Option<String>>>,
    pub fstype: String,
    pub uuid: String,
    pub partuuid: String,
    pub children: Vec<BlockDevice>,
}

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Disk {
    pub device: BlockDevice,
    pub sequence: String,
    pub blocked: String,
    pub removable: bool,
}

fn device_names(d: &BlockDevice) -> bool {
    // `^/dev/[a-zA-Z0-9_-]+$` and KNAME equality.
    let name_ok = d.name.starts_with("/dev/")
        && d.name.len() > 5
        && d.name[5..]
            .bytes()
            .all(|b| b.is_ascii_alphanumeric() || b == b'_' || b == b'-');
    if !name_ok || d.kname != d.name {
        return false;
    }
    // `^[0-9]+:[0-9]+$`
    match d.major_minor.split_once(':') {
        Some((major, minor)) => {
            !major.is_empty()
                && !minor.is_empty()
                && major.bytes().all(|b| b.is_ascii_digit())
                && minor.bytes().all(|b| b.is_ascii_digit())
        }
        None => false,
    }
}

fn unused_inventory(d: &BlockDevice) -> String {
    if !device_names(d) || d.size == 0 || d.mountpoints.is_none() {
        return "incomplete or unsupported device inventory".to_string();
    }
    if d.read_only {
        return "read-only device".to_string();
    }
    for point in d.mountpoints.as_ref().unwrap().iter().flatten() {
        if !point.is_empty() {
            return "mounted filesystem or active swap".to_string();
        }
    }
    unused_fstype(d)
}

fn unused_fstype(d: &BlockDevice) -> String {
    match d.fstype.as_str() {
        "iso9660" | "udf" => "installation/optical media".to_string(),
        "" | "ext2" | "ext3" | "ext4" | "xfs" | "vfat" | "exfat" | "ntfs" | "swap" => String::new(),
        _ => "unrecognized, multi-device or encrypted storage requires separate operator handling"
            .to_string(),
    }
}

fn unused_children(d: &BlockDevice) -> String {
    for child in &d.children {
        if child.device_type != "part" {
            return "active mapped/stacked device".to_string();
        }
        let reason = unused(child);
        if !reason.is_empty() {
            return reason;
        }
    }
    String::new()
}

fn unused(d: &BlockDevice) -> String {
    let reason = unused_inventory(d);
    if !reason.is_empty() {
        return reason;
    }
    unused_children(d)
}

fn decode_device(value: &JsonValue) -> Result<BlockDevice, ()> {
    let soft = Soft::new(value)?;
    let mountpoints = match soft.array("mountpoints")? {
        None => None,
        Some(items) => {
            let mut out = Vec::new();
            for item in items {
                match item {
                    JsonValue::Null => out.push(None),
                    JsonValue::Str(s) => out.push(Some(s.clone())),
                    _ => return Err(()),
                }
            }
            Some(out)
        }
    };
    let mut children = Vec::new();
    if let Some(items) = soft.array("children")? {
        for item in items {
            children.push(decode_device(item)?);
        }
    }
    Ok(BlockDevice {
        name: soft.string("name")?.unwrap_or_default(),
        kname: soft.string("kname")?.unwrap_or_default(),
        device_type: soft.string("type")?.unwrap_or_default(),
        tran: soft.string("tran")?.unwrap_or_default(),
        size: soft.unsigned("size")?.unwrap_or(0),
        model: soft.string("model")?.unwrap_or_default(),
        serial: soft.string("serial")?.unwrap_or_default(),
        wwn: soft.string("wwn")?.unwrap_or_default(),
        major_minor: soft.string("maj:min")?.unwrap_or_default(),
        read_only: soft.boolean("ro")?.unwrap_or(false),
        mountpoints,
        fstype: soft.string("fstype")?.unwrap_or_default(),
        uuid: soft.string("uuid")?.unwrap_or_default(),
        partuuid: soft.string("partuuid")?.unwrap_or_default(),
        children,
    })
}

fn decode_tree(data: &[u8]) -> Result<Vec<BlockDevice>, Error> {
    let value = crate::jsongo::parse(data)
        .map_err(|_| Error::msg("cannot decode block device inventory"))?;
    // Go decodes `null` into a zero tree without error: no devices.
    if matches!(value, JsonValue::Null) {
        return Ok(Vec::new());
    }
    let soft = Soft::new(&value).map_err(|_| Error::msg("cannot decode block device inventory"))?;
    let items = soft
        .array("blockdevices")
        .map_err(|_| Error::msg("cannot decode block device inventory"))?;
    let mut devices = Vec::new();
    if let Some(items) = items {
        for item in items {
            devices.push(
                decode_device(item)
                    .map_err(|_| Error::msg("cannot decode block device inventory"))?,
            );
        }
    }
    Ok(devices)
}

pub fn disk_removable_at(sys_root: &str, d: &BlockDevice) -> bool {
    if fold_eq_ascii(&d.tran, "usb") {
        return true;
    }
    let path = pathx::join(sys_root, &[&pathx::base(&d.name), "removable"]);
    match std::fs::read(&path) {
        Ok(data) => go_trim_space(&String::from_utf8_lossy(&data)) == "1",
        Err(_) => false,
    }
}

/// `parentDiskOfSys`: resolve a `/dev` node to its parent disk via sysfs.
pub fn parent_disk_of_sys(sys_root: &str, dev: &str) -> String {
    let mut resolved = dev.to_string();
    if let Ok(canonical) = pathx::eval_symlinks(dev) {
        resolved = canonical;
    }
    if !resolved.starts_with("/dev/") {
        return String::new();
    }
    let link = match std::fs::read_link(pathx::join(sys_root, &[&pathx::base(&resolved)])) {
        Ok(link) => link,
        Err(_) => return String::new(),
    };
    let link = link.to_string_lossy().into_owned();
    let parent = pathx::base(&pathx::dir(&link));
    if parent != "block" {
        return format!("/dev/{parent}");
    }
    format!("/dev/{}", pathx::base(&resolved))
}

pub fn parent_disk_of(dev: &str) -> String {
    parent_disk_of_sys("/sys/class/block", dev)
}

/// Pure `parseLiveMediaDisks` over mount table and cmdline bytes.
pub fn parse_live_media_disks(
    mountinfo: &[u8],
    cmdline: &[u8],
    parent_of: &dyn Fn(&str) -> String,
) -> std::collections::BTreeSet<String> {
    let mut out = std::collections::BTreeSet::new();
    for line in String::from_utf8_lossy(mountinfo).split('\n') {
        let fields: Vec<&str> = line.split(' ').collect();
        let mut sep: Option<usize> = None;
        for (i, field) in fields.iter().enumerate() {
            if *field == "-" {
                sep = Some(i);
                break;
            }
        }
        let sep = match sep {
            Some(sep) => sep,
            None => continue,
        };
        if fields.len() < sep + 3 || fields.len() < 5 {
            continue;
        }
        let (mountpoint, fstype, source) = (fields[4], fields[sep + 1], fields[sep + 2]);
        let live_mount = mountpoint == "/run/initramfs/live"
            || mountpoint.starts_with("/run/media/")
            || mountpoint.starts_with("/run/archiso/");
        let live_fs = (fstype == "iso9660" || fstype == "udf") && source.starts_with("/dev/");
        if !live_mount && !live_fs {
            continue;
        }
        if !source.starts_with("/dev/") {
            continue;
        }
        let parent = parent_of(source);
        if !parent.is_empty() {
            out.insert(parent);
        }
    }
    for token in go_fields(&String::from_utf8_lossy(cmdline)) {
        let mut dev = "";
        for prefix in ["root=live:", "live:", "bootdev="] {
            if let Some(rest) = token.strip_prefix(prefix) {
                if rest.starts_with("/dev/") {
                    dev = rest;
                }
            }
        }
        if dev.is_empty() {
            continue;
        }
        let parent = parent_of(dev);
        if !parent.is_empty() {
            out.insert(parent);
        }
    }
    out
}

pub fn live_media_disks() -> std::collections::BTreeSet<String> {
    let mountinfo = std::fs::read("/proc/self/mountinfo").unwrap_or_default();
    let cmdline = std::fs::read("/proc/cmdline").unwrap_or_default();
    parse_live_media_disks(&mountinfo, &cmdline, &|dev| parent_disk_of(dev))
}

pub fn disk_sequence_at(sys_root: &str, device: &BlockDevice) -> Result<String, Error> {
    if !device_names(device) {
        return Err(Error::msg("invalid kernel device identity"));
    }
    let seq_path = pathx::join(sys_root, &[&pathx::base(&device.name), "diskseq"]);
    let data =
        std::fs::read(&seq_path).map_err(|e| crate::errors::path_error("open", &seq_path, e))?;
    let text = String::from_utf8_lossy(&data);
    let seq = go_trim_space(&text);
    if seq.is_empty() || !seq.bytes().all(|b| b.is_ascii_digit()) {
        return Err(Error::msg("invalid disk sequence"));
    }
    Ok(seq.to_string())
}

pub fn check_holders_at(sys_root: &str, device: &BlockDevice) -> Result<(), Error> {
    let entries = std::fs::read_dir(pathx::join(
        sys_root,
        &[&pathx::base(&device.name), "holders"],
    ))
    .map_err(|_| Error::msg(format!("cannot inspect device holders for {}", device.name)))?;
    // Go's `ReadDir` fails the whole inspection on any mid-stream error.
    let mut count = 0;
    for entry in entries {
        entry.map_err(|_| {
            Error::msg(format!("cannot inspect device holders for {}", device.name))
        })?;
        count += 1;
    }
    if count != 0 {
        return Err(Error::msg(format!(
            "device {} has holders; it is in use by another device",
            device.name
        )));
    }
    for child in &device.children {
        check_holders_at(sys_root, child)?;
    }
    Ok(())
}

/// `scanDisks` with injectable roots for tests; production passes the live
/// `/sys/class/block` and `live_media_disks()`.
pub fn scan_disks_at(
    ctx: &Ctx,
    run: &dyn Runner,
    sys_root: &str,
    media: &std::collections::BTreeSet<String>,
) -> Result<Vec<Disk>, Error> {
    let data = run
        .run(
            ctx,
            "lsblk",
            &[
                "--json".to_string(),
                "--bytes".to_string(),
                "--paths".to_string(),
                "--output".to_string(),
                "NAME,KNAME,TYPE,TRAN,SIZE,MODEL,SERIAL,WWN,MAJ:MIN,RO,MOUNTPOINTS,FSTYPE,UUID,PARTUUID".to_string(),
            ],
            None,
        )
        .map_err(|_| Error::msg("cannot inventory block devices"))?;
    let devices = decode_tree(&data)?;
    let mut disks = Vec::new();
    for device in devices {
        if device.device_type != "disk" {
            continue;
        }
        let mut disk = Disk {
            blocked: unused(&device),
            removable: disk_removable_at(sys_root, &device),
            device,
            sequence: String::new(),
        };
        if disk.blocked.is_empty() {
            match disk_sequence_at(sys_root, &disk.device) {
                Ok(sequence) => disk.sequence = sequence,
                Err(_) => disk.blocked = "cannot establish kernel disk identity".to_string(),
            }
            if disk.blocked.is_empty() {
                if let Err(err) = check_holders_at(sys_root, &disk.device) {
                    disk.blocked = err.to_string();
                }
            }
        }
        // The disk backing the running installer is never a target, even
        // when it looks like an ordinary writable disk.
        if media.contains(&disk.device.name) {
            disk.blocked = "current installer media".to_string();
        }
        disks.push(disk);
    }
    Ok(disks)
}

pub fn scan_disks(ctx: &Ctx, run: &dyn Runner) -> Result<Vec<Disk>, Error> {
    let media = live_media_disks();
    scan_disks_at(ctx, run, "/sys/class/block", &media)
}

pub fn same_disk(selected: &Disk, observed: &[Disk]) -> Result<(), Error> {
    for current in observed {
        if current.device.name == selected.device.name {
            if selected.sequence.is_empty() || selected != current {
                return Err(Error::msg(
                    "disk identity, partition inventory or use changed; no installation started",
                ));
            }
            return Ok(());
        }
    }
    Err(Error::msg(
        "selected disk disappeared; no installation started",
    ))
}

pub fn disk_summary(d: &BlockDevice) -> String {
    sprintf(
        "%q | %.1f GiB | model %q | serial %q | WWN %q",
        &[
            Arg::Str(&d.name),
            Arg::Float(d.size as f64 / (1u64 << 30) as f64),
            Arg::Str(&d.model),
            Arg::Str(&d.serial),
            Arg::Str(&d.wwn),
        ],
    )
}

#[cfg(test)]
mod tests;
