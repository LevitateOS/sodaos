# Retired OS display override

`os-release` is preserved as historical provenance only. It is not a build,
provisioning or installation input. Its sparse `/etc/os-release` replacement hid
upstream version metadata and broke native Fedora repository URL expansion.

The [installer branding guide](../../../docs/guides/media.md#sodaos-branding)
owns the corrected contract: retain the upstream metadata and brand supported
presentation surfaces instead.
