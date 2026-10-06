//! Floating Tailnet toolchain resolution: newest stable archive
//! selection, checksum/base-tag selection and resolve_tailnet_inputs
//! with the release-selection case.

use crate::coreos_stream::{tailnet_base_tags_url, tailnet_index_url};
use crate::files::oci_architecture;
use crate::http::{fetch_capped_json, fetch_capped_text, HttpTransport, UreqTransport};
use crate::json_go::Fields;
use crate::live_inputs::{valid_tailnet_inputs, TailnetInputs};
use crate::Error;
use soda_json::JsonValue;

/// Resolves the floating Tailnet toolchain: newest stable release, its
/// archive checksum, and the newest upstream alpine-base tag.
pub fn resolve_tailnet_inputs(arch: &str) -> Result<TailnetInputs, Error> {
    resolve_tailnet_inputs_with(&UreqTransport, arch)
}

pub fn resolve_tailnet_inputs_with<T: HttpTransport>(
    transport: &T,
    arch: &str,
) -> Result<TailnetInputs, Error> {
    let platform = oci_architecture(arch)?.to_string();
    let version = latest_tailnet_release_with(transport)?;
    let checksum_url = format!(
        "{index}tailscale_{version}_{platform}.tgz.sha256",
        index = tailnet_index_url()
    );
    let raw = fetch_capped_text(transport, &checksum_url, 1 << 20)?;
    let sha = raw.trim().split(' ').next().unwrap_or("").to_string();
    let base = latest_tailnet_base_tag_with(transport)?;
    let inputs = TailnetInputs {
        version,
        sha256: sha,
        base,
    };
    valid_tailnet_inputs(&inputs)?;
    Ok(inputs)
}

/// Scans for `tailscale_<major>.<minor>.<patch>_amd64.tgz` releases and
/// returns the newest version (original digit strings).
fn latest_tailnet_release(text: &str) -> Result<String, Error> {
    let bytes = text.as_bytes();
    let mut best: [i64; 3] = [0, 0, 0];
    let mut version = String::new();
    let mut i = 0;
    while i < bytes.len() {
        match parse_tailnet_release_at(bytes, i) {
            Some((end, parts)) => {
                let mut nums = [0i64; 3];
                for (slot, part) in parts.iter().enumerate() {
                    nums[slot] = part
                        .parse::<i64>()
                        .map_err(|_| Error::msg("unparseable Tailnet release"))?;
                }
                if version.is_empty()
                    || nums[0] > best[0]
                    || (nums[0] == best[0]
                        && (nums[1] > best[1] || (nums[1] == best[1] && nums[2] > best[2])))
                {
                    best = nums;
                    version = format!("{}.{}.{}", parts[0], parts[1], parts[2]);
                }
                i = end;
            }
            None => i += 1,
        }
    }
    if version.is_empty() {
        return Err(Error::msg("no Tailnet release in upstream index"));
    }
    Ok(version)
}

fn parse_tailnet_release_at(bytes: &[u8], start: usize) -> Option<(usize, [String; 3])> {
    let mut i = start;
    if bytes.get(i..i + 10)? != b"tailscale_" {
        return None;
    }
    i += 10;
    let mut parts = [String::new(), String::new(), String::new()];
    for (slot, part) in parts.iter_mut().enumerate() {
        let begin = i;
        while i < bytes.len() && bytes[i].is_ascii_digit() {
            i += 1;
        }
        if i == begin {
            return None;
        }
        *part = String::from_utf8_lossy(&bytes[begin..i]).into_owned();
        if slot < 2 {
            if bytes.get(i) != Some(&b'.') {
                return None;
            }
            i += 1;
        }
    }
    if bytes.get(i..i + 10)? != b"_amd64.tgz" {
        return None;
    }
    i += 10;
    Some((i, parts))
}

fn latest_tailnet_release_with<T: HttpTransport>(transport: &T) -> Result<String, Error> {
    let raw = fetch_capped_text(transport, &tailnet_index_url(), 1 << 20)?;
    latest_tailnet_release(&raw)
}

fn latest_tailnet_base_tag_with<T: HttpTransport>(transport: &T) -> Result<String, Error> {
    let data = fetch_capped_json(transport, &tailnet_base_tags_url(), 1 << 20)?;
    let text = String::from_utf8_lossy(&data);
    let value = JsonValue::parse(&text).map_err(|_| Error::msg("invalid Tailnet base tags"))?;
    let doc = Fields::of(&value).ok_or_else(|| Error::msg("invalid Tailnet base tags"))?;
    let results = doc
        .object_list("Results")
        .map_err(|_| Error::msg("invalid Tailnet base tags"))?;
    let mut best = String::new();
    let mut best_v = [0i64; 2];
    for tag in &results {
        let name = tag
            .string("Name")
            .map_err(|_| Error::msg("invalid Tailnet base tags"))?;
        let (major_s, minor_s) = match name.split_once('.') {
            Some(pair) if !pair.0.is_empty() && !pair.1.is_empty() => pair,
            _ => continue,
        };
        if major_s.bytes().any(|b| !b.is_ascii_digit())
            || minor_s.bytes().any(|b| !b.is_ascii_digit())
        {
            continue;
        }
        if major_s.contains('.') || minor_s.contains('.') {
            continue;
        }
        // Go ignores Atoi errors here (overflow pins to 0).
        let major = major_s.parse::<i64>().unwrap_or(0);
        let minor = minor_s.parse::<i64>().unwrap_or(0);
        if best.is_empty() || major > best_v[0] || (major == best_v[0] && minor > best_v[1]) {
            best_v = [major, minor];
            best = format!("{major_s}.{minor_s}");
        }
    }
    if best.is_empty() {
        return Err(Error::msg("no Tailnet base tag upstream"));
    }
    Ok(format!("docker.io/tailscale/alpine-base:{best}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn oracle_tailnet_release_selection() {
        // Oracle: latestTailnetRelease newest-stable selection.
        let index =
            "tailscale_1.98.1_amd64.tgz\ntailscale_1.98.2_amd64.tgz\ntailscale_1.98.10_amd64.tgz\n";
        assert_eq!(latest_tailnet_release(index).unwrap(), "1.98.10");
        assert_eq!(
            latest_tailnet_release("nothing here")
                .unwrap_err()
                .message(),
            "no Tailnet release in upstream index"
        );
    }
}
