use std::fmt;

use serde::de::{DeserializeOwned, Error as DeError, IgnoredAny, MapAccess, Visitor};
use serde::{Deserialize, Deserializer};

use crate::document::read_document;
use crate::fetch::{discover, fetch_document, verify_releases};
use crate::model::{admit_channel, Channel, Highwater, Permit, Seen, Trust};
use crate::native::Runner;
use crate::{is_digest_ref, now_unix, Error};

use super::{upload, Ledger};

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
struct TagList {
    repository: String,
    tags: Vec<String>,
}
impl<'de> Deserialize<'de> for TagList {
    fn deserialize<D: Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct V;
        impl<'de> Visitor<'de> for V {
            type Value = TagList;
            fn expecting(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                f.write_str("a tag-list object")
            }
            fn visit_map<M: MapAccess<'de>>(self, mut m: M) -> Result<TagList, M::Error> {
                let (mut repository, mut tags) = (None, None);
                while let Some(key) = m.next_key::<String>()? {
                    match key.as_str() {
                        "Repository" => {
                            repository = Some(m.next_value::<Box<serde_json::value::RawValue>>()?)
                        }
                        "Tags" => tags = Some(m.next_value::<Box<serde_json::value::RawValue>>()?),
                        _ => {
                            m.next_value::<IgnoredAny>()?;
                        }
                    }
                }
                Ok(TagList {
                    repository: decode_raw(repository)?,
                    tags: decode_raw(tags)?,
                })
            }
        }
        d.deserialize_map(V)
    }
}

#[cfg(test)]
mod json_slot_tests {
    use super::TagList;

    #[test]
    fn tag_list_uses_final_exact_values_before_typed_conversion() {
        let tags: TagList = serde_json::from_str(
            r#"{"Repository":false,"Repository":"repo","Tags":["old"],"Tags":null,"unknown":true}"#,
        )
        .unwrap();
        assert_eq!(tags.repository, "repo");
        assert!(tags.tags.is_empty());
        assert!(serde_json::from_str::<TagList>(
            r#"{"Repository":"repo","Tags":null,"Tags":[false]}"#
        )
        .is_err());
    }
}

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
    let offer: Channel = read_document(copy, &p.digest)?;
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
    let listed: TagList = serde_json::from_slice(&tags).map_err(|_| Error::refused())?;
    if listed.repository != repository || listed.tags.is_empty() {
        return Err(Error::refused());
    }
    Ok(listed.tags.iter().any(|name| name == role))
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
    let old: Channel = fetch_document(r, t, &previous, &format!("{out}/previous-channel"))?;
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
