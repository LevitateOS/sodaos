use serde::Serialize;

fn pretty_json(value: &impl Serialize) -> String {
    serde_json::to_string_pretty(value).expect("serializing a JSON record cannot fail")
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct WorkerRecord<'a> {
    Executable: &'a str,
    Source: &'a str,
    ForgejoSource: &'a str,
    OutputParent: &'a str,
    StorageRoot: &'a str,
    BuildHome: &'a str,
    Runtime: &'a str,
    Tools: &'a str,
    MediaAuthorityDirectory: &'a str,
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct TrustRecord<'a> {
    Format: u8,
    Prefix: &'a str,
    Epoch: u8,
    Keys: TrustKeys<'a>,
    NotBefore: u64,
    MaxAgeSeconds: u16,
    ClockSkewSeconds: u8,
    MinimumSequence: MinimumSequence,
}

#[derive(Serialize)]
struct TrustKeys<'a> {
    artifact: [&'a str; 1],
    candidate: [&'a str; 1],
    preview: [&'a str; 1],
    stable: [&'a str; 1],
}

#[derive(Serialize)]
struct MinimumSequence {
    candidate: u8,
    preview: u8,
    stable: u8,
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct ConfigRecord {
    Trust: &'static str,
    Keys: ConfigKeys,
}

#[derive(Serialize)]
#[allow(non_snake_case)]
struct ConfigKeys {
    Key: &'static str,
    Passphrase: &'static str,
}

/// `worker_json` emits the worker configuration record without a final LF.
#[allow(clippy::too_many_arguments)]
pub(super) fn worker_json(
    executable: &str,
    source: &str,
    forgejo_source: &str,
    output_parent: &str,
    storage_root: &str,
    build_home: &str,
    runtime: &str,
    tools: &str,
    authority: &str,
) -> String {
    pretty_json(&WorkerRecord {
        Executable: executable,
        Source: source,
        ForgejoSource: forgejo_source,
        OutputParent: output_parent,
        StorageRoot: storage_root,
        BuildHome: build_home,
        Runtime: runtime,
        Tools: tools,
        MediaAuthorityDirectory: authority,
    })
}

/// `trust_json` emits the trust record with its required final LF.
pub(super) fn trust_json(prefix: &str, now: u64, pubs: [&str; 4]) -> String {
    let record = TrustRecord {
        Format: 1,
        Prefix: prefix,
        Epoch: 1,
        Keys: TrustKeys {
            artifact: [pubs[0]],
            candidate: [pubs[1]],
            preview: [pubs[2]],
            stable: [pubs[3]],
        },
        NotBefore: now - 600,
        MaxAgeSeconds: 3600,
        ClockSkewSeconds: 10,
        MinimumSequence: MinimumSequence {
            candidate: 1,
            preview: 1,
            stable: 1,
        },
    };
    pretty_json(&record) + "\n"
}

/// `config_json` emits the authority configuration record with its required final LF.
pub(super) fn config_json() -> String {
    pretty_json(&ConfigRecord {
        Trust: "/run/soda-media-authority/trust.json",
        Keys: ConfigKeys {
            Key: "/run/soda-media-authority/artifact.private",
            Passphrase: "/run/soda-media-authority/passphrase",
        },
    }) + "\n"
}
