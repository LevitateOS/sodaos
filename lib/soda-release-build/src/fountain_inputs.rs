//! Admission of one independently produced Fountain release and source proofs.
use crate::http::{get_follow, HttpTransport, UreqTransport};
use crate::{coreos::https_url, Error};
use serde::Deserialize;
use sha2::{Digest, Sha256};
use soda_build_tools::payload::{FountainRelease, ReleaseFile};
use std::io::Read;
use std::time::Duration;

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Metadata {
    schema: u32,
    version: String,
    source_commit: String,
    source_archive: String,
    source_sha256: String,
    prepared_source: String,
    prepared_source_sha256: String,
    image: String,
    image_config_digest: String,
    platform: String,
    extension_protocol: u32,
    browser_contract: u32,
}

pub fn resolve_fountain_release(
    url: &str,
    expected_revision: &str,
) -> Result<FountainRelease, Error> {
    resolve_fountain_release_with(&UreqTransport, url, expected_revision)
}

fn fetch_fountain_metadata<T: HttpTransport>(
    transport: &T,
    url: &str,
) -> Result<(Vec<u8>, String), Error> {
    if !https_url(url) || !url.ends_with("/release.json") {
        return Err(Error::msg(
            "explicit immutable HTTPS Fountain release.json URL required",
        ));
    }
    let mut response = get_follow(
        transport,
        url,
        None,
        Duration::from_secs(300),
        "unsafe Fountain metadata redirect",
    )?;
    if response.status != 200 {
        return Err(Error::msg("Fountain release metadata unavailable"));
    }
    let mut bytes = Vec::new();
    response
        .body
        .by_ref()
        .take((1 << 20) + 1)
        .read_to_end(&mut bytes)
        .map_err(|_| Error::msg("Fountain metadata unreadable"))?;
    if bytes.is_empty() || bytes.len() > 1 << 20 {
        return Err(Error::msg("Fountain metadata exceeds limit"));
    }
    let prefix = url
        .strip_suffix("release.json")
        .ok_or_else(|| Error::msg("explicit immutable HTTPS Fountain release.json URL required"))?
        .to_owned();
    Ok((bytes, prefix))
}

fn admit_fountain_metadata(
    metadata: &Metadata,
    prefix: &str,
    expected_revision: &str,
) -> Result<(), Error> {
    if metadata.source_commit != expected_revision {
        return Err(Error::msg(
            "Fountain metadata differs from selected source revision",
        ));
    }
    if metadata.schema != 1
        || metadata.platform != "linux/amd64"
        || !prefix.ends_with(&format!("/v{}/", metadata.version))
        || metadata.source_archive != format!("fountain-{}-source.tar.gz", metadata.version)
        || !metadata
            .prepared_source
            .starts_with(&format!("forgejo-src-{}+", metadata.version))
        || !metadata.prepared_source.ends_with(".tar.gz")
        || !metadata
            .prepared_source
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || b"._+-".contains(&byte))
    {
        return Err(Error::msg(
            "Fountain release identity, platform or source filenames disagree",
        ));
    }
    Ok(())
}

pub fn resolve_fountain_release_with<T: HttpTransport>(
    transport: &T,
    url: &str,
    expected_revision: &str,
) -> Result<FountainRelease, Error> {
    let (bytes, prefix) = fetch_fountain_metadata(transport, url)?;
    let metadata: Metadata = serde_json::from_slice(&bytes)
        .map_err(|_| Error::msg("invalid Fountain release metadata"))?;
    admit_fountain_metadata(&metadata, &prefix, expected_revision)?;
    let mut release = FountainRelease {
        version: metadata.version.clone(),
        metadata: ReleaseFile {
            url: url.into(),
            bytes: bytes.len() as u64,
            sha256: format!("{:x}", Sha256::digest(&bytes)),
        },
        image: metadata.image,
        image_config: metadata.image_config_digest,
        source_revision: metadata.source_commit,
        source: ReleaseFile::default(),
        prepared_source: ReleaseFile::default(),
        extension_protocol: metadata.extension_protocol,
        browser_contract: metadata.browser_contract,
    };
    release
        .validate_identity()
        .map_err(|error| Error::msg(error.to_string()))?;
    release.source = source_file(
        transport,
        &format!("{prefix}{}", metadata.source_archive),
        &metadata.source_sha256,
    )?;
    release.prepared_source = source_file(
        transport,
        &format!("{prefix}{}", metadata.prepared_source),
        &metadata.prepared_source_sha256,
    )?;
    release
        .validate()
        .map_err(|error| Error::msg(error.to_string()))?;
    Ok(release)
}

fn source_file<T: HttpTransport>(
    transport: &T,
    url: &str,
    sha256: &str,
) -> Result<ReleaseFile, Error> {
    if !https_url(url)
        || sha256.len() != 64
        || !sha256
            .bytes()
            .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
    {
        return Err(Error::msg("invalid Fountain source digest"));
    }
    let response = get_follow(
        transport,
        url,
        None,
        Duration::from_secs(300),
        "unsafe Fountain source redirect",
    )?;
    if response.status != 200 {
        return Err(Error::msg("Fountain corresponding source unavailable"));
    }
    let mut body = response.body.take((2u64 << 30) + 1);
    let mut buffer = [0; 65536];
    let mut size = 0;
    let mut hash = Sha256::new();
    loop {
        let n = body
            .read(&mut buffer)
            .map_err(|_| Error::msg("Fountain source download failed"))?;
        if n == 0 {
            break;
        }
        size += n as u64;
        if size > 2u64 << 30 {
            return Err(Error::msg("Fountain source exceeds limit"));
        }
        hash.update(&buffer[..n]);
    }
    if size == 0 || format!("{:x}", hash.finalize()) != sha256 {
        return Err(Error::msg("Fountain source digest mismatch"));
    }
    Ok(ReleaseFile {
        url: url.into(),
        bytes: size,
        sha256: sha256.into(),
    })
}
