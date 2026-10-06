//! `publish.go`: ledger-serialized publication with observation.

use std::collections::BTreeMap;

use soda_json::JsonValue;

use crate::buildx::fresh_directory;
use crate::document::read_json;
use crate::fetch::{discover, lock_state, save_state};
use crate::model::{Channel, Permit, Trust};
use crate::native::{private_file, verify_copy, write_json, Runner};
use crate::prepare::immutable_tag;
use crate::{is_channel, is_digest_ref, now_unix, Error};

mod ledger;
pub use ledger::{init_ledger, Ledger};

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

mod channel;
use channel::{admit_channel_offer, promote_channel};

#[allow(clippy::too_many_arguments)] // Arity mirrors the Go owner 1:1.
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
    upload(
        r,
        t,
        reference,
        copy,
        &immutable_tag(reference),
        auth,
        &immutable,
    )?;
    verify_copy(
        r,
        t,
        reference,
        &format!("docker://{reference}"),
        &format!("{out}/registry-roundtrip"),
    )
}

#[allow(clippy::too_many_arguments)] // Arity mirrors the Go owner 1:1.
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
#[allow(clippy::too_many_arguments)] // Arity mirrors the Go owner 1:1.
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
        promote_channel(
            r, t, p, &role, &reference, &copy, auth, out, &offer, &ledger,
        )?;
    }
    finalize_publication(r, t, p, &role, &reference, ledger_path, out, &ledger)
}

#[cfg(test)]
mod tests {
    use super::*;

    use super::channel::validate_channel_history;
    use crate::model::{empty_state, Seen};

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
