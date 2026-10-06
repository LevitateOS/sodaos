pub(super) mod metadata;

pub(super) use metadata::{
    parse_oci_config, parse_oci_manifest, read_oci_index, OciBlobData, OciDescriptor, OciImage,
};
