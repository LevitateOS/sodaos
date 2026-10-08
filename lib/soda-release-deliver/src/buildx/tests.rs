use super::*;

#[test]
fn toolchain_validation_matches_go() {
    let good = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: ["build-base-0.5-r4", "gcc-14.2.0-r6", "musl-dev-1.2.5-r10"]
            .iter()
            .map(|s| s.to_string())
            .collect(),
    };
    assert!(good.validate().is_ok());
    let bad = ForgejoToolchain {
        compiler_image: "other".to_string(),
        apk_packages: good.apk_packages.clone(),
    };
    assert_eq!(
        bad.validate().unwrap_err(),
        Error::msg("invalid Forgejo compiler provenance")
    );
    let unsorted = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: ["gcc-14.2.0-r6", "build-base-0.5-r4"]
            .iter()
            .map(|s| s.to_string())
            .collect(),
    };
    assert!(unsorted.validate().is_err());
    let missing = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: ["build-base-0.5-r4".to_string()].to_vec(),
    };
    assert_eq!(
        missing.validate().unwrap_err(),
        Error::msg("incomplete Forgejo APK provenance")
    );
}

#[test]
fn file_helpers_match_go_errors() {
    assert_eq!(
        fresh_directory("relative/path").unwrap_err(),
        Error::msg("absolute new directory required")
    );
    assert_eq!(
        private_destination("relative").unwrap_err(),
        Error::msg("absolute private output required")
    );
    let dir = std::env::temp_dir().join(format!("srd-{}", std::process::id()));
    std::fs::create_dir_all(&dir).unwrap();
    let target = dir.join("fresh");
    fresh_directory(target.to_str().unwrap()).unwrap();
    assert!(fresh_directory(target.to_str().unwrap()).is_err());
    std::fs::remove_dir_all(&dir).unwrap();
}

#[test]
fn rooted_file_helpers_refuse_intermediate_directory_symlink_escape() {
    let temp = tempfile::tempdir().unwrap();
    let root_path = temp.path().join("root");
    let outside_path = temp.path().join("outside");
    std::fs::create_dir(&root_path).unwrap();
    std::fs::create_dir(&outside_path).unwrap();
    std::fs::write(outside_path.join("presentation.json"), b"{}").unwrap();
    std::os::unix::fs::symlink("../outside", root_path.join("forgejo-context")).unwrap();

    let root = Root::open(root_path.to_str().unwrap()).unwrap();
    let name = "forgejo-context/presentation.json";
    let refused = [
        ("hash_at", hash_at(&root, name).is_err()),
        ("read_at", read_at(&root, name, 1024).is_err()),
        ("open_layout_entry", open_layout_entry(&root, name).is_err()),
        ("read_json_at", read_json_at(&root, name).is_err()),
    ];
    assert!(
        refused.iter().all(|(_, refused)| *refused),
        "intermediate symlink escape was accepted: {refused:?}"
    );
}

#[test]
fn rooted_file_helpers_allow_internal_directory_links_and_preserve_file_contracts() {
    use sha2::{Digest as _, Sha256};

    let temp = tempfile::tempdir().unwrap();
    let root_path = temp.path().join("root");
    std::fs::create_dir(&root_path).unwrap();
    let context_path = root_path.join("context-data");
    let blobs_path = root_path.join("blob-data");
    std::fs::create_dir(&context_path).unwrap();
    std::fs::create_dir(&blobs_path).unwrap();
    std::fs::create_dir(blobs_path.join("sha256")).unwrap();
    std::os::unix::fs::symlink("context-data", root_path.join("forgejo-context")).unwrap();
    std::os::unix::fs::symlink("blob-data", root_path.join("blobs")).unwrap();

    let json = format!("{{}}{}", " ".repeat((4 << 20) - 2));
    std::fs::write(context_path.join("presentation.json"), &json).unwrap();
    let oversized_json = format!("{json} ");
    std::fs::write(context_path.join("oversized.json"), oversized_json).unwrap();
    let blob = b"raw\0blob\xffbytes";
    let digest = format!("{:x}", Sha256::digest(blob));
    std::fs::write(blobs_path.join("sha256").join(&digest), blob).unwrap();

    let root = Root::open(root_path.to_str().unwrap()).unwrap();
    let presentation = "forgejo-context/presentation.json";
    let (payload, json_digest) = read_json_at(&root, presentation).unwrap();
    assert_eq!(payload, crate::payload::Payload::default());
    assert_eq!(
        json_digest,
        format!("{:x}", Sha256::digest(json.as_bytes()))
    );
    assert_eq!(
        read_at(&root, presentation, 4 << 20).unwrap().len(),
        4 << 20
    );
    assert!(read_json_at(&root, "forgejo-context/oversized.json").is_err());

    let blob_name = format!("blobs/sha256/{digest}");
    assert_eq!(hash_at(&root, &blob_name).unwrap(), digest);
    assert_eq!(read_at(&root, &blob_name, blob.len() as i64).unwrap(), blob);
    assert!(read_at(&root, &blob_name, blob.len() as i64 - 1).is_err());
    let (mut layout_file, layout_size) = open_layout_entry(&root, &blob_name).unwrap();
    let mut layout_bytes = Vec::new();
    std::io::Read::read_to_end(&mut layout_file, &mut layout_bytes).unwrap();
    assert_eq!(layout_bytes, blob);
    assert_eq!(layout_size, blob.len() as i64);
}

#[test]
fn rooted_file_helpers_refuse_final_symlinks_and_nonregular_files() {
    let temp = tempfile::tempdir().unwrap();
    let root_path = temp.path().join("root");
    std::fs::create_dir(&root_path).unwrap();
    std::fs::create_dir(root_path.join("forgejo-context")).unwrap();
    std::fs::write(root_path.join("forgejo-context/target.json"), b"{}").unwrap();
    std::os::unix::fs::symlink(
        "target.json",
        root_path.join("forgejo-context/presentation.json"),
    )
    .unwrap();
    let fifo = root_path.join("forgejo-context/pipe");
    use std::os::unix::ffi::OsStrExt;
    let fifo_c = std::ffi::CString::new(fifo.as_os_str().as_bytes()).unwrap();
    // SAFETY: mkfifo receives a live NUL-terminated path and a conventional mode.
    assert_eq!(unsafe { libc::mkfifo(fifo_c.as_ptr(), 0o600) }, 0);

    let root = Root::open(root_path.to_str().unwrap()).unwrap();
    for name in ["\0", "/absolute", "../escape"] {
        assert_eq!(hash_at(&root, name).unwrap_err(), Error::refused());
        assert_eq!(read_at(&root, name, 1024).unwrap_err(), Error::refused());
        assert_eq!(
            open_layout_entry(&root, name).unwrap_err(),
            Error::refused()
        );
        assert_eq!(read_json_at(&root, name).unwrap_err(), Error::refused());
    }
    for name in ["forgejo-context/presentation.json", "forgejo-context/pipe"] {
        assert!(hash_at(&root, name).is_err(), "hash_at accepted {name}");
        assert!(
            read_at(&root, name, 1024).is_err(),
            "read_at accepted {name}"
        );
        assert!(
            open_layout_entry(&root, name).is_err(),
            "open_layout_entry accepted {name}"
        );
        assert!(
            read_json_at(&root, name).is_err(),
            "read_json_at accepted {name}"
        );
    }
}
