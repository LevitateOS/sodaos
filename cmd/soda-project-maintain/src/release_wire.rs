use super::release::ReleasePayload;

pub(crate) fn decode_release_payload(body: &[u8]) -> Result<ReleasePayload, ()> {
    use soda_release_deliver::payload::Role;
    let payload = soda_build_tools::payload::PayloadV4::parse(body).map_err(|_| ())?;
    Ok(ReleasePayload {
        architecture: payload.architecture.clone(),
        project_os: Role::ProjectOs.binding(&payload.images).config.clone(),
        tailscale: payload
            .optional_extensions
            .tailnet
            .images
            .tailscale
            .config
            .clone(),
    })
}
