//! Block-device inventory: `lsblk` decoding, use/refusal checks, kernel
//! identity (`diskseq`, holders), live-media exclusion, and summaries.

use crate::command::Runner;
use crate::errors::Error;
use crate::pathx;
use crate::signal::Ctx;
use serde::de::{IgnoredAny, MapAccess, SeqAccess, Visitor};
use serde::{Deserialize, Deserializer};
use serde_json::value::RawValue;
use std::fmt;

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

#[derive(Default)]
struct RawBlockRoot {
    blockdevices: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawBlockRoot {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawBlockRoot;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an lsblk root object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawBlockRoot, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawBlockRoot::default();
                while let Some(key) = map.next_key::<String>()? {
                    if key == "blockdevices" {
                        dto.blockdevices = Some(map.next_value()?);
                    } else {
                        let _: IgnoredAny = map.next_value()?;
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

struct RawBlockList(Vec<Box<RawValue>>);
impl<'de> Deserialize<'de> for RawBlockList {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawBlockList;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an array of block devices")
            }
            fn visit_seq<A>(self, mut seq: A) -> Result<RawBlockList, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut items = Vec::new();
                while let Some(item) = seq.next_element::<Box<RawValue>>()? {
                    items.push(item);
                }
                Ok(RawBlockList(items))
            }
        }
        d.deserialize_seq(V)
    }
}

#[derive(Default)]
struct RawBlockDevice {
    name: Option<Box<RawValue>>,
    kname: Option<Box<RawValue>>,
    kind: Option<Box<RawValue>>,
    tran: Option<Box<RawValue>>,
    size: Option<Box<RawValue>>,
    model: Option<Box<RawValue>>,
    serial: Option<Box<RawValue>>,
    wwn: Option<Box<RawValue>>,
    major_minor: Option<Box<RawValue>>,
    ro: Option<Box<RawValue>>,
    mountpoints: Option<Box<RawValue>>,
    fstype: Option<Box<RawValue>>,
    uuid: Option<Box<RawValue>>,
    partuuid: Option<Box<RawValue>>,
    children: Option<Box<RawValue>>,
}
impl<'de> Deserialize<'de> for RawBlockDevice {
    fn deserialize<D>(d: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = RawBlockDevice;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a block-device object")
            }
            fn visit_map<A>(self, mut map: A) -> Result<RawBlockDevice, A::Error>
            where
                A: MapAccess<'de>,
            {
                let mut dto = RawBlockDevice::default();
                while let Some(key) = map.next_key::<String>()? {
                    match key.as_str() {
                        "name" => dto.name = Some(map.next_value()?),
                        "kname" => dto.kname = Some(map.next_value()?),
                        "type" => dto.kind = Some(map.next_value()?),
                        "tran" => dto.tran = Some(map.next_value()?),
                        "size" => dto.size = Some(map.next_value()?),
                        "model" => dto.model = Some(map.next_value()?),
                        "serial" => dto.serial = Some(map.next_value()?),
                        "wwn" => dto.wwn = Some(map.next_value()?),
                        "maj:min" => dto.major_minor = Some(map.next_value()?),
                        "ro" => dto.ro = Some(map.next_value()?),
                        "mountpoints" => dto.mountpoints = Some(map.next_value()?),
                        "fstype" => dto.fstype = Some(map.next_value()?),
                        "uuid" => dto.uuid = Some(map.next_value()?),
                        "partuuid" => dto.partuuid = Some(map.next_value()?),
                        "children" => dto.children = Some(map.next_value()?),
                        _ => {
                            let _: IgnoredAny = map.next_value()?;
                        }
                    }
                }
                Ok(dto)
            }
        }
        d.deserialize_map(V)
    }
}

fn block_string(raw: Option<&RawValue>) -> Result<String, ()> {
    match raw {
        None => Ok(String::new()),
        Some(raw) if raw.get() == "null" => Ok(String::new()),
        Some(raw) => serde_json::from_str(raw.get()).map_err(|_| ()),
    }
}
fn block_size(raw: Option<&RawValue>) -> Result<u64, ()> {
    match raw {
        None => Ok(0),
        Some(raw) if raw.get() == "null" => Ok(0),
        Some(raw) => raw
            .get()
            .parse::<i128>()
            .ok()
            .and_then(|n| u64::try_from(n).ok())
            .ok_or(()),
    }
}
fn decode_device(raw: &RawValue) -> Result<BlockDevice, ()> {
    let dto: RawBlockDevice = serde_json::from_str(raw.get()).map_err(|_| ())?;
    let size = block_size(dto.size.as_deref())?;
    let read_only = match dto.ro.as_deref() {
        None => false,
        Some(raw) if raw.get() == "null" => false,
        Some(raw) => serde_json::from_str(raw.get()).map_err(|_| ())?,
    };
    let mountpoints = match dto.mountpoints.as_deref() {
        None => None,
        Some(raw) if raw.get() == "null" => None,
        Some(raw) => Some(serde_json::from_str::<Vec<Option<String>>>(raw.get()).map_err(|_| ())?),
    };
    let mut children = Vec::new();
    if let Some(raw) = dto.children.as_deref() {
        if raw.get() != "null" {
            let items: RawBlockList = serde_json::from_str(raw.get()).map_err(|_| ())?;
            for item in items.0 {
                children.push(decode_device(&item)?);
            }
        }
    }
    Ok(BlockDevice {
        name: block_string(dto.name.as_deref())?,
        kname: block_string(dto.kname.as_deref())?,
        device_type: block_string(dto.kind.as_deref())?,
        tran: block_string(dto.tran.as_deref())?,
        size,
        model: block_string(dto.model.as_deref())?,
        serial: block_string(dto.serial.as_deref())?,
        wwn: block_string(dto.wwn.as_deref())?,
        major_minor: block_string(dto.major_minor.as_deref())?,
        read_only,
        mountpoints,
        fstype: block_string(dto.fstype.as_deref())?,
        uuid: block_string(dto.uuid.as_deref())?,
        partuuid: block_string(dto.partuuid.as_deref())?,
        children,
    })
}

fn decode_tree(data: &[u8]) -> Result<Vec<BlockDevice>, Error> {
    let text = String::from_utf8_lossy(data);
    let mut de = serde_json::Deserializer::from_str(&text);
    let raw = Box::<RawValue>::deserialize(&mut de)
        .map_err(|_| Error::msg("cannot decode block device inventory"))?;
    de.end()
        .map_err(|_| Error::msg("cannot decode block device inventory"))?;
    if raw.get() == "null" {
        return Ok(Vec::new());
    }
    let dto: RawBlockRoot = serde_json::from_str(raw.get())
        .map_err(|_| Error::msg("cannot decode block device inventory"))?;
    let mut devices = Vec::new();
    if let Some(raw) = dto.blockdevices {
        if raw.get() != "null" {
            let items: RawBlockList = serde_json::from_str(raw.get())
                .map_err(|_| Error::msg("cannot decode block device inventory"))?;
            for item in items.0 {
                devices.push(
                    decode_device(&item)
                        .map_err(|_| Error::msg("cannot decode block device inventory"))?,
                );
            }
        }
    }
    Ok(devices)
}

pub fn disk_removable_at(sys_root: &str, d: &BlockDevice) -> bool {
    if d.tran.eq_ignore_ascii_case("usb") {
        return true;
    }
    let path = pathx::join(sys_root, &[&pathx::base(&d.name), "removable"]);
    match std::fs::read(&path) {
        Ok(data) => String::from_utf8_lossy(&data).trim() == "1",
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
    for token in String::from_utf8_lossy(cmdline).split_whitespace() {
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
    let seq = text.trim();
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
    format!(
        "{:?} | {:.1} GiB | model {:?} | serial {:?} | WWN {:?}",
        d.name,
        d.size as f64 / (1u64 << 30) as f64,
        d.model,
        d.serial,
        d.wwn,
    )
}

#[cfg(test)]
mod tests;
