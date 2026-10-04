//! `publish.go`: ledger-serialized publication with observation.

use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::buildx::{fresh_directory, private_destination};
use crate::document::{read_document, read_json};
use crate::fetch::{discover, fetch_document, lock_state, save_state, verify_releases};
use crate::jsonx::{parse_lenient, Binder};
use crate::model::{
    admit_channel, empty_state, Channel, Highwater, Permit, Seen, Trust,
};
use crate::native::{private_file, verify_copy, write_json, Runner};
use crate::payload::{decode_opt_i64, decode_opt_string};
use crate::prepare::immutable_tag;
use crate::{is_channel, is_digest_ref, now_unix, Error};

/// `Ledger`: persistent per-repository publication state.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Ledger {
    pub format: i64,
    pub repository: String,
    pub phase: String,
    pub digest: String,
    pub state: Highwater,
}

impl Ledger {
    pub fn validate(&self, t: &Trust) -> Result<(), Error> {
        if self.format != 1 || self.state.validate().is_err() || self.state.trust_epoch > t.epoch
        {
            return Err(Error::refused());
        }
        t.role(&self.repository)?;
        if !valid_ledger_phase(&self.phase, &self.digest) {
            return Err(Error::refused());
        }
        Ok(())
    }

    pub fn decode(value: &JsonValue) -> Result<Ledger, String> {
        let mut b = Binder::new(value).map_err(|_| "invalid ledger".to_string())?;
        let mut ledger = Ledger {
            format: decode_opt_i64(&mut b, "Format")?,
            repository: decode_opt_string(&mut b, "Repository")?,
            phase: decode_opt_string(&mut b, "Phase")?,
            digest: decode_opt_string(&mut b, "Digest")?,
            state: Highwater::default(),
        };
        if let Some(entries) = b
            .entries("State")
            .map_err(|_| "invalid field State".to_string())?
        {
            ledger.state = Highwater::decode(&JsonValue::Object(entries.to_vec()))?;
        }
        b.finish_name()?;
        Ok(ledger)
    }
}

impl crate::jsonx::Emit for Ledger {
    fn emit(&self, e: &mut crate::jsonx::Emitter) {
        e.begin_object(false);
        e.field(true, "Format");
        e.int(self.format);
        e.field(false, "Repository");
        e.string(&self.repository);
        e.field(false, "Phase");
        e.string(&self.phase);
        e.field(false, "Digest");
        e.string(&self.digest);
        e.field(false, "State");
        self.state.emit(e);
        e.end_object(false);
    }
}

/// `InitLedger`: initialize a per-repository publication ledger.
pub fn init_ledger(path: &str, t: &Trust, repo: &str) -> Result<(), Error> {
    t.validate()?;
    t.role(repo)?;
    private_destination(path)?;
    let mut state = empty_state();
    state.trust_epoch = t.epoch;
    write_json(
        path,
        &Ledger {
            format: 1,
            repository: repo.to_string(),
            phase: "idle".to_string(),
            state,
            ..Ledger::default()
        },
    )
}

fn valid_ledger_phase(phase: &str, digest: &str) -> bool {
    match phase {
        "idle" => digest.is_empty(),
        "pending" | "complete" => is_digest_ref(digest),
        _ => false,
    }
}

fn upload(
    r: &dyn Runner,
    t: &Trust,
    reference: &str,
    signed: &str,
    dest: &str,
    auth: &str,
    out: &str,
) -> Result<(), Error> {
    let (repo, _) = t.reference(reference)?;
    let policy = policy_for_upload(t, &repo, signed)?;
    let policy_path = format!("{out}/upload-policy.json");
    write_json(&policy_path, &policy)?;
    let registry = registry_config_for_upload(out, t)?;
    r.run(&[
        "--command-timeout=10m",
        "--policy",
        &policy_path,
        "--registries.d",
        &registry,
        "copy",
        "--preserve-digests",
        "--src-no-creds",
        "--dest-authfile",
        auth,
        &format!("dir:{signed}"),
        &format!("docker://{dest}"),
    ])?;
    Ok(())
}

// Policy/registry helpers shared with native.rs shapes.
fn policy_for_upload(t: &Trust, repo: &str, signed: &str) -> Result<JsonValue, Error> {
    crate::native::policy_for_publish(t, repo, "dir", signed)
}

fn registry_config_for_upload(out: &str, t: &Trust) -> Result<String, Error> {
    crate::native::registry_config_for_publish(out, t)
}

fn observe_published(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    role: &str,
    out: &str,
) -> Result<(), Error> {
    let reference = format!("{}@{}", p.repository, p.digest);
    if is_channel(role) {
        let actual = discover(r, t, role)?;
        if actual != reference {
            return Err(Error::msg(
                "publication not observed at intended channel; held, not replayed",
            ));
        }
    }
    verify_copy(
        r,
        t,
        &reference,
        &format!("docker://{reference}"),
        &format!("{out}/observed"),
    )
}

fn admit_publish_ledger(
    t: &Trust,
    p: &Permit,
    ledger_path: &str,
    out: &str,
) -> Result<(String, Ledger), Error> {
    t.validate()?;
    let role = t.role(&p.repository)?;
    if p.format != 1 || !is_digest_ref(&p.digest) {
        return Err(Error::refused());
    }
    let ledger: Ledger = read_json(ledger_path, Ledger::decode)?;
    if ledger.validate(t).is_err() || ledger.repository != p.repository {
        return Err(Error::refused());
    }
    fresh_directory(out)?;
    Ok((role, ledger))
}

fn should_observe(observe: bool, ledger: &Ledger, p: &Permit) -> bool {
    observe || (ledger.phase == "complete" && ledger.digest == p.digest)
}

fn observe_only(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    role: &str,
    ledger_path: &str,
    out: &str,
    ledger: &Ledger,
) -> Result<(), Error> {
    if ledger.digest != p.digest || ledger.phase == "idle" {
        return Err(Error::refused());
    }
    observe_published(r, t, p, role, out)?;
    let mut ledger = ledger.clone();
    ledger.phase = "complete".to_string();
    ledger.state.trust_epoch = t.epoch;
    save_state(ledger_path, &ledger)?;
    let mut receipt = BTreeMap::new();
    receipt.insert(
        "Reference".to_string(),
        format!("{}@{}", p.repository, p.digest),
    );
    receipt.insert(
        "Outcome".to_string(),
        "publication observed; no write replayed".to_string(),
    );
    write_json(&format!("{out}/receipt.json"), &receipt)
}

fn validate_channel_history(current: &Seen, previous: &str) -> Result<(), Error> {
    if previous != "absent" && !is_digest_ref(previous) {
        return Err(Error::refused());
    }
    if current.sequence == 0 && previous != "absent" {
        return Err(Error::msg(
            "existing channel requires preserved publisher history; no implicit adoption",
        ));
    }
    if current.sequence != 0 && previous != current.digest {
        return Err(Error::msg("protected permit disagrees with publisher history"));
    }
    Ok(())
}

fn admit_channel_offer(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    role: &str,
    copy: &str,
    out: &str,
    current: &Highwater,
) -> Result<(Highwater, Channel), Error> {
    let empty = Seen::default();
    let seen = current.channels.get(role).unwrap_or(&empty);
    validate_channel_history(seen, &p.previous)?;
    let offer: Channel = read_document(copy, &p.digest, Channel::decode)?;
    let next = admit_channel(t, current, &offer, &p.digest, role, now_unix())?;
    let (verified, result) = verify_releases(r, t, &next, &offer, "", out);
    result?;
    Ok((verified, offer))
}

fn admit_signed(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    role: &str,
    signed: &str,
    auth: &str,
    ledger_path: &str,
    out: &str,
    ledger: &Ledger,
) -> Result<(String, Channel, Ledger), Error> {
    if p.validate(t, now_unix()).is_err() || private_file(auth).is_err() {
        return Err(Error::refused());
    }
    let reference = format!("{}@{}", p.repository, p.digest);
    let check = format!("{out}/signed-check");
    verify_copy(r, t, &reference, &format!("dir:{signed}"), &check)?;
    let copy = format!("{check}/image");
    let mut offer = Channel::default();
    let mut next = ledger.state.clone();
    if is_channel(role) {
        let (verified, admitted) = admit_channel_offer(r, t, p, role, &copy, out, &next)?;
        next = verified;
        offer = admitted;
    }
    next.trust_epoch = t.epoch;
    let mut ledger = ledger.clone();
    ledger.phase = "pending".to_string();
    ledger.digest = p.digest.clone();
    ledger.state = next;
    save_state(ledger_path, &ledger)?;
    Ok((copy, offer, ledger))
}

fn commit_immutable(
    r: &dyn Runner,
    t: &Trust,
    reference: &str,
    copy: &str,
    auth: &str,
    out: &str,
) -> Result<(), Error> {
    let immutable = format!("{out}/immutable-upload");
    {
        use std::os::unix::fs::DirBuilderExt;
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&immutable)
            .map_err(|e| Error::msg(format!("mkdir {immutable}: {e}")))?;
    }
    upload(r, t, reference, copy, &immutable_tag(reference), auth, &immutable)?;
    verify_copy(
        r,
        t,
        reference,
        &format!("docker://{reference}"),
        &format!("{out}/registry-roundtrip"),
    )
}

fn channel_tag_exists(
    r: &dyn Runner,
    repository: &str,
    role: &str,
) -> Result<bool, Error> {
    let tags = r.run(&[
        "--command-timeout=2m",
        "list-tags",
        "--no-creds",
        &format!("docker://{repository}"),
    ])?;
    let value = parse_lenient(&tags).map_err(|_| Error::refused())?;
    let soft = crate::jsonx::Soft::new(&value).map_err(|_| Error::refused())?;
    let listed = soft.string("Repository").map_err(|_| Error::refused())?.unwrap_or_default();
    let names = soft.array("Tags").map_err(|_| Error::refused())?.unwrap_or(&[]);
    if listed != repository || names.is_empty() {
        return Err(Error::refused());
    }
    for tag in names {
        match tag {
            JsonValue::Str(name) if name == role => return Ok(true),
            JsonValue::Str(_) => {}
            _ => return Err(Error::refused()),
        }
    }
    Ok(false)
}

fn verify_previous_channel(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    role: &str,
    out: &str,
    offer: &Channel,
) -> Result<(), Error> {
    let previous = discover(r, t, role)?;
    let digest = previous.split('@').nth(1).unwrap_or("");
    if digest != p.previous {
        return Err(Error::msg("channel changed since protected admission"));
    }
    let old: Channel = fetch_document(
        r,
        t,
        &previous,
        &format!("{out}/previous-channel"),
        Channel::decode,
    )?;
    if old.format != 1 || old.name != role || old.sequence >= offer.sequence || old.issued > offer.issued
    {
        return Err(Error::refused());
    }
    Ok(())
}

fn promote_channel(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    role: &str,
    reference: &str,
    copy: &str,
    auth: &str,
    out: &str,
    offer: &Channel,
    ledger: &Ledger,
) -> Result<(), Error> {
    let exists = channel_tag_exists(r, &p.repository, role)?;
    if exists {
        verify_previous_channel(r, t, p, role, out, offer)?;
    } else if p.previous != "absent" {
        return Err(Error::msg("expected channel missing; no automatic bootstrap"));
    }
    p.validate(t, now_unix())?;
    admit_channel(t, &ledger.state, offer, &p.digest, role, now_unix())?;
    let promotion = format!("{out}/promotion");
    {
        use std::os::unix::fs::DirBuilderExt;
        std::fs::DirBuilder::new()
            .mode(0o700)
            .create(&promotion)
            .map_err(|e| Error::msg(format!("mkdir {promotion}: {e}")))?;
    }
    upload(
        r,
        t,
        reference,
        copy,
        &format!("{}:{role}", p.repository),
        auth,
        &promotion,
    )
}

fn finalize_publication(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    role: &str,
    reference: &str,
    ledger_path: &str,
    out: &str,
    ledger: &Ledger,
) -> Result<(), Error> {
    observe_published(r, t, p, role, out)?;
    let mut ledger = ledger.clone();
    ledger.phase = "complete".to_string();
    save_state(ledger_path, &ledger)?;
    let mut receipt = BTreeMap::new();
    receipt.insert("Reference".to_string(), reference.to_string());
    receipt.insert(
        "Outcome".to_string(),
        "published and anonymously signature-verified; no activation".to_string(),
    );
    write_json(&format!("{out}/receipt.json"), &receipt)
}

/// `Publish`: ledger-serialized publication for a protected worker.
pub fn publish(
    r: &dyn Runner,
    t: &Trust,
    p: &Permit,
    signed: &str,
    auth: &str,
    ledger_path: &str,
    out: &str,
    observe: bool,
) -> Result<(), Error> {
    let _lock = lock_state(ledger_path)?;
    let (role, ledger) = admit_publish_ledger(t, p, ledger_path, out)?;
    if should_observe(observe, &ledger, p) {
        return observe_only(r, t, p, &role, ledger_path, out, &ledger);
    }
    if ledger.phase == "pending" {
        return Err(Error::msg(
            "uncertain publication held; use observation, never blind replay",
        ));
    }
    let reference = format!("{}@{}", p.repository, p.digest);
    let (copy, offer, ledger) =
        admit_signed(r, t, p, &role, signed, auth, ledger_path, out, &ledger)?;
    commit_immutable(r, t, &reference, &copy, auth, out)?;
    if is_channel(&role) {
        promote_channel(r, t, p, &role, &reference, &copy, auth, out, &offer, &ledger)?;
    }
    finalize_publication(r, t, p, &role, &reference, ledger_path, out, &ledger)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn trust() -> Trust {
        Trust {
            prefix: "ghcr.io/example/sodaos".to_string(),
            ..Trust::default()
        }
    }

    #[test]
    fn ledger_phases() {
        let trust = trust();
        let idle = Ledger {
            format: 1,
            repository: format!("{}-release", trust.prefix),
            phase: "idle".to_string(),
            state: empty_state(),
            ..Ledger::default()
        };
        assert!(idle.validate(&trust).is_ok());
        let mut pending = idle.clone();
        pending.phase = "pending".to_string();
        pending.digest = format!("sha256:{}", "a".repeat(64));
        assert!(pending.validate(&trust).is_ok());
        let mut bad = idle.clone();
        bad.phase = "bogus".to_string();
        assert_eq!(bad.validate(&trust).unwrap_err(), Error::refused());
        let mut wrong_repo = idle.clone();
        wrong_repo.repository = "ghcr.io/other/x".to_string();
        assert_eq!(wrong_repo.validate(&trust).unwrap_err(), Error::refused());
    }

    #[test]
    fn channel_history_rules() {
        let empty = Seen::default();
        assert!(validate_channel_history(&empty, "absent").is_ok());
        assert!(validate_channel_history(&empty, &format!("sha256:{}", "a".repeat(64))).is_err());
        let seen = Seen {
            sequence: 2,
            digest: format!("sha256:{}", "b".repeat(64)),
            issued: 10,
        };
        assert!(validate_channel_history(&seen, &format!("sha256:{}", "b".repeat(64))).is_ok());
        assert!(validate_channel_history(&seen, "absent").is_err());
    }
}
