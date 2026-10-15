//! Current candidate metadata admission shared by checks and release staging.
use crate::buildx::read_at;
use crate::buildx::Root;
use crate::model::Candidate;
use crate::payload::Payload;
use crate::Error;

pub fn load_admitted_candidate(candidate: &str) -> Result<(Payload, Candidate, Vec<u8>), Error> {
    let root = Root::open(candidate)?;
    let payload = read_at(&root, "payload.json", 1 << 20)?;
    let raw = read_at(&root, "candidate.json", 1 << 20)?;
    let p: Payload =
        Payload::parse(&payload).map_err(|_| Error::msg("candidate metadata refused"))?;
    let c: Candidate =
        crate::json_serde::strict(&raw).map_err(|_| Error::msg("candidate metadata refused"))?;
    c.validate(&p, &payload)?;
    Ok((p, c, payload))
}
