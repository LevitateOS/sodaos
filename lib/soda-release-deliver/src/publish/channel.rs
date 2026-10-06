use soda_json::JsonValue;

use crate::document::read_document;
use crate::fetch::{discover, fetch_document, verify_releases};
use crate::jsonx::parse_lenient;
use crate::model::{admit_channel, Channel, Highwater, Permit, Seen, Trust};
use crate::native::Runner;
use crate::{is_digest_ref, now_unix, Error};

use super::{upload, Ledger};

pub(super) fn validate_channel_history(current: &Seen, previous: &str) -> Result<(), Error> {
    if previous != "absent" && !is_digest_ref(previous) {
        return Err(Error::refused());
    }
    if current.sequence == 0 && previous != "absent" {
        return Err(Error::msg(
            "existing channel requires preserved publisher history; no implicit adoption",
        ));
    }
    if current.sequence != 0 && previous != current.digest {
        return Err(Error::msg(
            "protected permit disagrees with publisher history",
        ));
    }
    Ok(())
}

#[allow(clippy::too_many_arguments)] // Arity mirrors the Go owner 1:1.
pub(super) fn admit_channel_offer(
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

fn channel_tag_exists(r: &dyn Runner, repository: &str, role: &str) -> Result<bool, Error> {
    let tags = r.run(&[
        "--command-timeout=2m",
        "list-tags",
        "--no-creds",
        &format!("docker://{repository}"),
    ])?;
    let value = parse_lenient(&tags).map_err(|_| Error::refused())?;
    let soft = crate::jsonx::Soft::new(&value).map_err(|_| Error::refused())?;
    let listed = soft
        .string("Repository")
        .map_err(|_| Error::refused())?
        .unwrap_or_default();
    let names = soft
        .array("Tags")
        .map_err(|_| Error::refused())?
        .unwrap_or(&[]);
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
    if old.format != 1
        || old.name != role
        || old.sequence >= offer.sequence
        || old.issued > offer.issued
    {
        return Err(Error::refused());
    }
    Ok(())
}

#[allow(clippy::too_many_arguments)] // Arity mirrors the Go owner 1:1.
pub(super) fn promote_channel(
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
        return Err(Error::msg(
            "expected channel missing; no automatic bootstrap",
        ));
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
