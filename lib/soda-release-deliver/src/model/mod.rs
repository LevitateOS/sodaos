//! `model.go`: candidate, release and Docker metadata.

mod candidate;
pub(crate) use candidate::path_clean;
pub use candidate::{valid_candidate_content, Candidate};

pub(crate) mod docker;
pub use docker::{BuildPurpose, Distribution, DockerMetadata, FileIdentity, DOCKER_REPOSITORY};

mod release;
pub(crate) use release::valid_media_binding;
pub use release::{MediaBinding, MediaFile};
