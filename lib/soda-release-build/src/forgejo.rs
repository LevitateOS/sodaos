//! Archived Forgejo fork builds (`forgejo.go`): pinned compiler
//! provenance, musl binary builds, and image-context staging.

use crate::files::{is_revision, oci_architecture, write_new};
use crate::json_go::{marshal_indent, Emit, Strict};
use crate::production::Production;
use crate::{io_error, Error};
use std::path::{Path, PathBuf};

pub use soda_build_tools::reader::forgejo::{
    has_native_build_tools, valid_apk_list, FORGEJO_BUN_SHA256, FORGEJO_BUN_VERSION,
    FORGEJO_COMPAT_TOKEN, FORGEJO_COMPILER_IMAGE, FORGEJO_UPSTREAM_BASE,
};

/// Maps the exact archived revision to the Makefile-shaped version
/// identity: `<base>-soda.<short>+<compat>`.
pub fn forgejo_version_stamp(revision: &str) -> Result<String, Error> {
    if !is_revision(revision) {
        return Err(Error::msg("exact Forgejo source revision required"));
    }
    Ok(format!(
        "{FORGEJO_UPSTREAM_BASE}-soda.{}+{FORGEJO_COMPAT_TOKEN}",
        &revision[..12]
    ))
}

/// Pinned Forgejo compiler provenance record.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ForgejoToolchain {
    pub compiler_image: String,
    pub apk_packages: Vec<String>,
}

impl ForgejoToolchain {
    pub fn validate(&self) -> Result<(), Error> {
        if self.compiler_image != FORGEJO_COMPILER_IMAGE || !valid_apk_list(&self.apk_packages) {
            return Err(Error::msg("invalid Forgejo compiler provenance"));
        }
        if !has_native_build_tools(&self.apk_packages) {
            return Err(Error::msg("incomplete Forgejo APK provenance"));
        }
        Ok(())
    }

    pub fn marshal(&self) -> String {
        marshal_indent(&Emit::Object(vec![
            (
                "CompilerImage".to_string(),
                Emit::Str(self.compiler_image.clone()),
            ),
            (
                "APKPackages".to_string(),
                Emit::List(self.apk_packages.iter().cloned().map(Emit::Str).collect()),
            ),
        ]))
    }

    pub fn decode(binder: &mut Strict<'_>) -> Result<ForgejoToolchain, String> {
        Ok(ForgejoToolchain {
            compiler_image: binder.string("CompilerImage")?,
            apk_packages: binder.string_list("APKPackages")?,
        })
    }
}

pub fn record_forgejo_toolchain(prod: &Production, root: &Path) -> Result<(), Error> {
    let data = std::fs::read(root.join("apk-packages.txt"))
        .map_err(|e| io_error("open", &root.join("apk-packages.txt"), e))?;
    let text = String::from_utf8_lossy(&data);
    let record = ForgejoToolchain {
        compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
        apk_packages: text.trim().split('\n').map(str::to_string).collect(),
    };
    record.validate()?;
    write_new(
        &PathBuf::from(&prod.out).join("forgejo-toolchain.json"),
        (record.marshal() + "\n").as_bytes(),
        0o600,
    )
}

/// Compiles the exact archived fork under musl.
pub fn build_forgejo_binary(prod: &Production) -> Result<String, Error> {
    validate_forgejo_build(prod)?;
    prod.step("Build patched Forgejo binary")?;
    let root = PathBuf::from(&prod.native).join("forgejo-build");
    std::fs::create_dir(&root).map_err(|e| io_error("mkdir", &root, e))?;
    crate::files::chmod(&root, 0o700)?;
    let platform = oci_architecture(&prod.arch)?.to_string();
    let args = forgejo_build_args(prod, &platform);
    prod.call_execute(&prod.forgejo_source, "podman", &args)?;
    inspect_forgejo_build(prod, &root)
}

fn validate_forgejo_build(prod: &Production) -> Result<(), Error> {
    if !Path::new(&prod.forgejo_source).is_absolute() || prod.execute.is_none() {
        return Err(Error::msg("explicit archived Forgejo source required"));
    }
    Ok(())
}

pub fn forgejo_build_args(prod: &Production, platform: &str) -> Vec<String> {
    // Production callers always set ForgejoRevision; the archive gate
    // rejects anything else before this build runs. An empty stamp keeps
    // upstream defaults for direct unit-test callers only.
    let stamp = forgejo_version_stamp(&prod.forgejo_revision).unwrap_or_default();
    let root = PathBuf::from(&prod.native).join("forgejo-build");
    let script = format!(
        "set -eu\n\
         apk add --no-cache build-base=0.5-r4\n\
         wget -O /tmp/bun.zip \"https://github.com/oven-sh/bun/releases/download/bun-v{FORGEJO_BUN_VERSION}/bun-linux-x64-musl.zip\"\n\
         echo \"{FORGEJO_BUN_SHA256}  /tmp/bun.zip\" | sha256sum -c -\n\
         rm -rf /tmp/bun-extract && mkdir -p /tmp/bun-extract\n\
         unzip -q -o -d /tmp/bun-extract /tmp/bun.zip\n\
         install -m 755 /tmp/bun-extract/bun-linux-x64-musl/bun /usr/local/bin/bun\n\
         test \"$(bun --version)\" = \"{FORGEJO_BUN_VERSION}\"\n\
         mkdir -p /work/out/gocache /work/out/gopath /work/out/tmp\n\
         LC_ALL=C apk info -v | LC_ALL=C sort > /work/out/apk-packages.txt\n\
         # The git archive never contains built frontend outputs; generate them with\n\
         # the pinned Bun toolchain before bindata embeds public/. The frozen install\n\
         # resolves from the archived package lock; Bun is the only JS toolchain.\n\
         bun install --frozen-lockfile --no-progress\n\
         BROWSERSLIST_IGNORE_OLD_DATA=true bun ./node_modules/.bin/webpack\n\
         test -s public/assets/js/index.js\n\
         test -s public/assets/css/index.css\n\
         for package in options public templates migration; do\n\
         \u{20} (cd modules/$package && go generate -tags bindata .)\n\
         done\n\
         LDFLAGS=\"\"\n\
         if [ -n \"${{FORGEJO_VERSION:-}}\" ]; then\n\
         \u{20} LDFLAGS=\"-X main.Version=${{FORGEJO_VERSION}} -X main.ForgejoVersion=${{FORGEJO_VERSION}} -X main.ReleaseVersion=${{FORGEJO_VERSION}} -X \\\"main.Tags=bindata sqlite sqlite_unlock_notify\\\"\"\n\
         fi\n\
         go build -buildvcs=false -tags 'bindata sqlite sqlite_unlock_notify' -ldflags \"${{LDFLAGS}}\" -trimpath -o /work/out/forgejo-bin ."
    );
    vec![
        "--remote=false".to_string(),
        "run".to_string(),
        "--rm".to_string(),
        "--pull=always".to_string(),
        format!("--platform=linux/{platform}"),
        format!("--volume={}:/work/source:Z", prod.forgejo_source),
        format!("--volume={}:/work/out:Z", root.display()),
        "--workdir=/work/source".to_string(),
        "--env=GOCACHE=/work/out/gocache".to_string(),
        "--env=GOPATH=/work/out/gopath".to_string(),
        "--env=TMPDIR=/work/out/tmp".to_string(),
        format!("--env=FORGEJO_VERSION={stamp}"),
        FORGEJO_COMPILER_IMAGE.to_string(),
        "sh".to_string(),
        "-ec".to_string(),
        script,
    ]
}

fn inspect_forgejo_build(prod: &Production, root: &Path) -> Result<String, Error> {
    record_forgejo_toolchain(prod, root)?;
    let binary = root.join("forgejo-bin");
    crate::files::chmod(&binary, 0o755)?;
    crate::elf::inspect_elf(&binary, &prod.arch)?;
    Ok(binary.to_string_lossy().into_owned())
}

/// Builds the fork binary and hard-links it into this build's context.
pub fn stage_fork_binary(prod: &Production, context: &str) -> Result<(), Error> {
    let binary = build_forgejo_binary(prod)?;
    let prefix = format!("{}{}", prod.out, std::path::MAIN_SEPARATOR);
    if !context.starts_with(&prefix) {
        return Err(Error::msg(
            "forgejo image context must belong to this build",
        ));
    }
    std::fs::hard_link(&binary, Path::new(context).join("forgejo-bin"))
        .map_err(|e| io_error("link", Path::new(context), e))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::production::Production;

    #[test]
    fn oracle_version_stamp_vectors() {
        // Oracle: forgejoVersionStamp outputs.
        assert_eq!(
            forgejo_version_stamp(&"a".repeat(40)).unwrap(),
            "15.0.9-soda.aaaaaaaaaaaa+gitea-1.22.0"
        );
        assert_eq!(
            forgejo_version_stamp("dirty").unwrap_err().message(),
            "exact Forgejo source revision required"
        );
    }

    #[test]
    fn oracle_toolchain_provenance_vectors() {
        // Oracle: ForgejoToolchain.Validate outcomes.
        let good = ForgejoToolchain {
            compiler_image: FORGEJO_COMPILER_IMAGE.to_string(),
            apk_packages: vec![
                "build-base-0.5-r4".to_string(),
                "gcc-14.2.0-r6".to_string(),
                "musl-dev-1.2.5-r10".to_string(),
            ],
        };
        good.validate().unwrap();
        let bad_image = ForgejoToolchain {
            compiler_image: "docker.io/library/golang:1.26.7-alpine".to_string(),
            ..good.clone()
        };
        assert_eq!(
            bad_image.validate().unwrap_err().message(),
            "invalid Forgejo compiler provenance"
        );
        let unsorted = ForgejoToolchain {
            apk_packages: vec!["gcc-14.2.0-r6".to_string(), "build-base-0.5-r4".to_string()],
            ..good.clone()
        };
        assert_eq!(
            unsorted.validate().unwrap_err().message(),
            "invalid Forgejo compiler provenance"
        );
        let incomplete = ForgejoToolchain {
            apk_packages: vec!["build-base-0.5-r4".to_string()],
            ..good.clone()
        };
        assert_eq!(
            incomplete.validate().unwrap_err().message(),
            "incomplete Forgejo APK provenance"
        );
    }

    #[test]
    fn oracle_stage_fork_binary_wiring() {
        // Oracle: TestStageForkBinaryUsesExplicitFrozenSource.
        let root = std::env::temp_dir().join(format!(
            "soda-forgejo-{}-{}",
            std::process::id(),
            std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .unwrap()
                .as_nanos()
        ));
        let source = root.join("frozen-fork");
        let out = root.join("output");
        let context = out.join("forgejo-context");
        let native = root.join("native");
        for dir in [&source, &context, &native] {
            std::fs::create_dir_all(dir).unwrap();
        }
        let seen = std::sync::Arc::new(std::sync::Mutex::new(Vec::<String>::new()));
        let seen_clone = seen.clone();
        let source_clone = source.clone();
        let native_clone = native.clone();
        let mut prod = Production::default();
        prod.forgejo_source = source.to_string_lossy().into_owned();
        prod.native = native_clone.to_string_lossy().into_owned();
        prod.out = out.to_string_lossy().into_owned();
        prod.arch = "x86_64".to_string();
        prod.next = Some(Box::new(|_| Ok(())));
        prod.execute = Some(Box::new(move |dir, name, args| {
            assert_eq!(dir, source_clone.to_string_lossy());
            assert_eq!(name, "podman");
            *seen_clone.lock().unwrap() = args.to_vec();
            std::fs::write(
                native.join("forgejo-build/apk-packages.txt"),
                b"build-base-0.5-r4\ngcc-14.2.0-r6\nmusl-dev-1.2.5-r10\n",
            )
            .unwrap();
            std::fs::write(
                native.join("forgejo-build/forgejo-bin"),
                crate::elf::tests::fixture_elf(),
            )
            .unwrap();
            Ok(())
        }));
        stage_fork_binary(&prod, &context.to_string_lossy()).unwrap();
        let joined = seen.lock().unwrap().join(" ");
        for required in [
            "--pull=always",
            "--platform=linux/amd64",
            &format!("--volume={}:/work/source:Z", source.display()),
            FORGEJO_COMPILER_IMAGE,
            "apk add --no-cache build-base=0.5-r4",
            "apk info -v",
            "go generate -tags bindata",
            "go build -buildvcs=false -tags 'bindata sqlite sqlite_unlock_notify'",
        ] {
            assert!(joined.contains(required), "missing {required}");
        }
        assert!(!joined.contains("golang:1.26.7-alpine"));
        assert!(!joined.contains("apk add --no-cache build-base git"));
        assert!(context.join("forgejo-bin").exists());
        let toolchain: ForgejoToolchain = crate::json_input::read_json(
            &out.join("forgejo-toolchain.json"),
            "build.ForgejoToolchain",
            ForgejoToolchain::decode,
        )
        .unwrap();
        toolchain.validate().unwrap();
    }
}
