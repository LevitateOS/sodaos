//! `finalize.go`: protected final release signing after qualification.

use soda_json::JsonValue;

use crate::admission::admit_qualification;
use crate::buildx::{fresh_directory, write_new};
use crate::document::{read_json, write_document};
use crate::jsonx::{marshal, parse_lenient, Binder, Emit, Emitter, Soft};
use crate::model::{Channel, Permit, Trust};
use crate::native::{private_file, sign, Runner, SecretFiles};
use crate::payload::{decode_opt_string, decode_opt_u64};
use crate::prepare::{prepare, reference_for_document, Qualification};
use crate::publish::publish;
use crate::{is_digest_ref, now_unix, Error};

/// `Config`: operator admission for final signing.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Config {
    pub trust: String,
    pub signer: String,
    pub serial: u64,
    pub class: String,
    pub notes: String,
    pub auth_file: String,
    pub ledger: String,
    pub channel: String,
    pub channel_signer: String,
}

impl Config {
    pub fn decode(value: &JsonValue) -> Result<Config, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid config".to_string())?;
        let config = Config {
            trust: decode_opt_string(&mut b, "Trust")?,
            signer: decode_opt_string(&mut b, "Signer")?,
            serial: decode_opt_u64(&mut b, "Serial")?,
            class: decode_opt_string(&mut b, "Class")?,
            notes: decode_opt_string(&mut b, "Notes")?,
            auth_file: decode_opt_string(&mut b, "AuthFile")?,
            ledger: decode_opt_string(&mut b, "Ledger")?,
            channel: decode_opt_string(&mut b, "Channel")?,
            channel_signer: decode_opt_string(&mut b, "ChannelSigner")?,
        };
        b.finish_name()?;
        Ok(config)
    }
}

impl Emit for Config {
    fn emit(&self, e: &mut Emitter) {
        e.begin_object(false);
        e.field(true, "Trust");
        e.string(&self.trust);
        e.field(false, "Signer");
        e.string(&self.signer);
        e.field(false, "Serial");
        e.uint(self.serial);
        e.field(false, "Class");
        e.string(&self.class);
        e.field(false, "Notes");
        e.string(&self.notes);
        e.field(false, "AuthFile");
        e.string(&self.auth_file);
        e.field(false, "Ledger");
        e.string(&self.ledger);
        e.field(false, "Channel");
        e.string(&self.channel);
        e.field(false, "ChannelSigner");
        e.string(&self.channel_signer);
        e.end_object(false);
    }
}

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
    let config: Config = read_json(path, Config::decode)?;
    admit_config(&config)?;
    Ok(config)
}

fn load_trust(path: &str) -> Result<Trust, Error> {
    let trust: Trust = read_json(path, Trust::decode)
        .map_err(|_| Error::msg("public release trust configuration refused"))?;
    if trust.validate().is_err() {
        return Err(Error::msg("public release trust configuration refused"));
    }
    Ok(trust)
}

fn load_signer(path: &str) -> Result<SecretFiles, Error> {
    let keys: SecretFiles =
        read_json(path, SecretFiles::decode).map_err(|_| Error::msg("restricted signer configuration refused"))?;
    if private_file(&keys.key).is_err() || private_file(&keys.passphrase).is_err() {
        return Err(Error::msg("restricted signer key inputs required"));
    }
    Ok(keys)
}

fn document_digest(oci: &str) -> Result<String, Error> {
    let data = std::fs::read(format!("{oci}/index.json"))
        .map_err(|e| Error::msg(format!("read {oci}/index.json: {e}")))?;
    let value = parse_lenient(&data).map_err(|_| Error::refused())?;
    let soft = Soft::new(&value).map_err(|_| Error::refused())?;
    let manifests = soft.array("manifests").map_err(|_| Error::refused())?.unwrap_or(&[]);
    if manifests.len() != 1 {
        return Err(Error::refused());
    }
    let manifest = Soft::new(&manifests[0]).map_err(|_| Error::refused())?;
    let digest = manifest.string("digest").map_err(|_| Error::refused())?.unwrap_or_default();
    if !is_digest_ref(&digest) {
        return Err(Error::refused());
    }
    Ok(digest)
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
    let mut offer: Channel = read_json(&c.channel, Channel::decode)?;
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

fn write_final_receipt(out: &str, reference: &str, digest: &str, published: bool) -> Result<(), Error> {
    // Go map order is sorted: digest, published, reference, scope.
    let receipt = JsonValue::Object(vec![
        ("digest".to_string(), JsonValue::Str(digest.to_string())),
        ("published".to_string(), JsonValue::Bool(published)),
        (
            "reference".to_string(),
            JsonValue::Str(reference.to_string()),
        ),
        (
            "scope".to_string(),
            JsonValue::Str("signed final release metadata".to_string()),
        ),
    ]);
    write_new(&format!("{out}/final.json"), &marshal(&receipt), 0o600)
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
    use crate::jsonx::{marshal, parse_strict};

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
        let bytes = marshal(&config);
        let value = parse_strict(&bytes).unwrap();
        assert_eq!(Config::decode(&value).unwrap(), config);
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
