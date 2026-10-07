//! `finalize.go`: protected final release signing after qualification.

use std::fmt;

use serde::de::{DeserializeOwned, Error as DeError, IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer, Serialize};

use crate::admission::admit_qualification;
use crate::buildx::{fresh_directory, write_new};
use crate::document::{read_json, write_document};
use crate::model::{Channel, Permit, Trust};
use crate::native::{private_file, sign, Runner, SecretFiles};
use crate::prepare::{prepare, reference_for_document, Qualification};
use crate::publish::publish;
use crate::{is_digest_ref, now_unix, Error};

fn decode_raw<T: DeserializeOwned + Default, E: DeError>(
    raw: Option<Box<serde_json::value::RawValue>>,
) -> Result<T, E> {
    let Some(raw) = raw else {
        return Ok(T::default());
    };
    serde_json::from_str::<Option<T>>(raw.get())
        .map_err(E::custom)
        .map(Option::unwrap_or_default)
}

#[derive(Default)]
struct IndexDocument {
    manifests: Vec<ManifestDescriptor>,
}
impl<'de> Deserialize<'de> for IndexDocument {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = IndexDocument;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI index object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<IndexDocument, M::Error> {
                let mut manifests = None;
                while let Some(key) = m.next_key::<String>()? {
                    if key == "manifests" {
                        manifests = Some(m.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        m.next_value::<IgnoredAny>()?;
                    }
                }
                Ok(IndexDocument {
                    manifests: decode_raw(manifests)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

#[derive(Default)]
struct ManifestDescriptor {
    digest: String,
}
impl<'de> Deserialize<'de> for ManifestDescriptor {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = ManifestDescriptor;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("an OCI manifest descriptor")
            }
            fn visit_map<M: MapAccess<'de>>(
                self,
                mut m: M,
            ) -> Result<ManifestDescriptor, M::Error> {
                let mut digest = None;
                while let Some(key) = m.next_key::<String>()? {
                    if key == "digest" {
                        digest = Some(m.next_value::<Box<serde_json::value::RawValue>>()?);
                    } else {
                        m.next_value::<IgnoredAny>()?;
                    }
                }
                Ok(ManifestDescriptor {
                    digest: decode_raw(digest)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

/// `Config`: operator admission for final signing.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default, deny_unknown_fields, rename_all = "PascalCase")]
pub struct Config {
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub trust: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub signer: String,
    #[serde(deserialize_with = "crate::json_serde::null_u64")]
    pub serial: u64,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub class: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub notes: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub auth_file: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub ledger: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub channel: String,
    #[serde(deserialize_with = "crate::json_serde::null_default")]
    pub channel_signer: String,
}

impl Config {}

fn admit_publication_inputs(c: &Config) -> Result<(), Error> {
    let mut set = 0;
    for path in [&c.auth_file, &c.ledger, &c.channel, &c.channel_signer] {
        if !path.is_empty() {
            set += 1;
        }
    }
    if set == 0 {
        return Ok(());
    }
    if set != 4 {
        return Err(Error::msg(
            "publication requires auth-file, ledger, channel and channel-signer together",
        ));
    }
    for path in [&c.auth_file, &c.ledger, &c.channel, &c.channel_signer] {
        private_file(path)?;
    }
    Ok(())
}

fn admit_config(c: &Config) -> Result<(), Error> {
    crate::admission::admit_release_identity(c)?;
    if std::fs::metadata(&c.trust).is_err() {
        return Err(Error::msg("public release trust configuration required"));
    }
    if private_file(&c.signer).is_err() {
        return Err(Error::msg("restricted signer configuration required"));
    }
    admit_publication_inputs(c)
}

/// `LoadConfig`: load protected operator admission (root only).
pub fn load_config(path: &str) -> Result<Config, Error> {
    if unsafe { libc::geteuid() } != 0 {
        return Err(Error::msg("protected finalization admission required"));
    }
    private_file(path)?;
    let config: Config = read_json(path)?;
    admit_config(&config)?;
    Ok(config)
}

fn load_trust(path: &str) -> Result<Trust, Error> {
    let trust: Trust =
        read_json(path).map_err(|_| Error::msg("public release trust configuration refused"))?;
    if trust.validate().is_err() {
        return Err(Error::msg("public release trust configuration refused"));
    }
    Ok(trust)
}

fn load_signer(path: &str) -> Result<SecretFiles, Error> {
    let keys: SecretFiles =
        read_json(path).map_err(|_| Error::msg("restricted signer configuration refused"))?;
    if private_file(&keys.key).is_err() || private_file(&keys.passphrase).is_err() {
        return Err(Error::msg("restricted signer key inputs required"));
    }
    Ok(keys)
}

fn document_digest(oci: &str) -> Result<String, Error> {
    let data = std::fs::read(format!("{oci}/index.json"))
        .map_err(|e| Error::msg(format!("read {oci}/index.json: {e}")))?;
    let index: IndexDocument = serde_json::from_slice(&data).map_err(|_| Error::refused())?;
    if index.manifests.len() != 1 {
        return Err(Error::refused());
    }
    let digest = &index.manifests[0].digest;
    if !is_digest_ref(&digest) {
        return Err(Error::refused());
    }
    Ok(digest.clone())
}

fn sign_digest(
    r: &dyn Runner,
    trust: &Trust,
    keys: &SecretFiles,
    repo: &str,
    prepared: &str,
    out: &str,
) -> Result<String, Error> {
    let digest = document_digest(prepared)?;
    let permit = Permit {
        format: 1,
        repository: repo.to_string(),
        digest: digest.clone(),
        previous: String::new(),
        expires: now_unix() + 3600,
    };
    sign(r, trust, &permit, "oci", prepared, out, keys)?;
    Ok(digest)
}

fn bind_channel_releases(offer: &mut Channel, release_ref: &str) -> Result<(), Error> {
    if offer.releases.is_empty() {
        return Err(Error::refused());
    }
    for reference in offer.releases.values_mut() {
        *reference = release_ref.to_string();
    }
    Ok(())
}

fn channel_repository(trust: &Trust, name: &str) -> Result<String, Error> {
    if name != "candidate" && name != "preview" && name != "stable" {
        return Err(Error::refused());
    }
    Ok(format!("{}-channel-{name}", trust.prefix))
}

fn publish_channel_last(
    r: &dyn Runner,
    trust: &Trust,
    c: &Config,
    release_ref: &str,
    out: &str,
) -> Result<(), Error> {
    let mut offer: Channel = read_json(&c.channel)?;
    bind_channel_releases(&mut offer, release_ref)?;
    let prepared = format!("{out}/channel-prepared");
    let digest = write_document(&prepared, &offer)?;
    let keys = load_signer(&c.channel_signer)?;
    let repo = channel_repository(trust, &offer.name)?;
    let signed = format!("{out}/channel-signed");
    sign_digest(r, trust, &keys, &repo, &prepared, &signed)?;
    let permit = Permit {
        format: 1,
        repository: repo,
        digest,
        previous: "absent".to_string(),
        expires: now_unix() + 3600,
    };
    publish(
        r,
        trust,
        &permit,
        &format!("{signed}/signed"),
        &c.auth_file,
        &c.ledger,
        &format!("{out}/publish"),
        false,
    )
}

fn write_final_receipt(
    out: &str,
    reference: &str,
    digest: &str,
    published: bool,
) -> Result<(), Error> {
    // Go map order is sorted: digest, published, reference, scope.
    let receipt = serde_json::json!({"digest": digest, "published": published, "reference": reference, "scope": "signed final release metadata"});
    let data = crate::document::marshal_go_pretty(&receipt)?;
    write_new(&format!("{out}/final.json"), &data, 0o600)
}

fn prepare_and_sign(
    r: &dyn Runner,
    trust: &Trust,
    keys: &SecretFiles,
    q: &Qualification,
    candidate: &str,
    media: &str,
    out: &str,
) -> Result<(String, String), Error> {
    let prepared = format!("{out}/prepared");
    let digest = prepare(trust, candidate, media, q, &prepared)?;
    let signed = format!("{out}/signed");
    let repo = format!("{}-release", trust.prefix);
    let got = sign_digest(r, trust, keys, &repo, &prepared, &signed)?;
    if got != digest {
        return Err(Error::refused());
    }
    let reference = reference_for_document(trust, "release", &digest)?;
    Ok((reference, digest))
}

/// `Finalize`: prepare and sign final release metadata from unchanged inputs.
pub fn finalize(
    r: &dyn Runner,
    c: &Config,
    candidate: &str,
    media: &str,
    evidence: &str,
    out: &str,
) -> Result<String, Error> {
    fresh_directory(out)?;
    let trust = load_trust(&c.trust)?;
    let keys = load_signer(&c.signer)?;
    let q = admit_qualification(c, candidate, media, evidence)?;
    let (reference, digest) = prepare_and_sign(r, &trust, &keys, &q, candidate, media, out)?;
    let published = !c.auth_file.is_empty();
    if published {
        publish_channel_last(r, &trust, c, &reference, out)?;
    }
    write_final_receipt(out, &reference, &digest, published)?;
    Ok(reference)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn publication_index_uses_final_exact_values_before_typed_conversion() {
        let index: IndexDocument = serde_json::from_str(
            r#"{"manifests":[{"digest":false,"digest":"sha256:ok"}],"manifests":[{"digest":"sha256:last"}],"future":1}"#,
        )
        .unwrap();
        assert_eq!(index.manifests.len(), 1);
        assert_eq!(index.manifests[0].digest, "sha256:last");
        assert!(serde_json::from_str::<ManifestDescriptor>(
            r#"{"digest":"sha256:ok","digest":false}"#
        )
        .is_err());
    }

    #[test]
    fn config_round_trip_and_admission() {
        let config = Config {
            trust: "/t/trust.json".to_string(),
            signer: "/t/signer.json".to_string(),
            serial: 9,
            class: "emergency".to_string(),
            notes: "n".to_string(),
            ..Config::default()
        };
        let bytes = crate::document::marshal_go_pretty(&config).unwrap();
        let decoded: Config = crate::json_serde::strict(&bytes).unwrap();
        assert_eq!(decoded, config);
        // Partial publication inputs refuse (after trust/signer admission).
        let mut partial = config.clone();
        partial.auth_file = "/t/auth".to_string();
        // trust path does not exist, so admission fails there first.
        assert_eq!(
            admit_config(&partial).unwrap_err(),
            Error::msg("public release trust configuration required")
        );
        if unsafe { libc::geteuid() } != 0 {
            assert_eq!(
                load_config("/nonexistent").unwrap_err(),
                Error::msg("protected finalization admission required")
            );
        }
    }
}
