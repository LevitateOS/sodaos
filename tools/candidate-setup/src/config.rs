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
    FountainReleaseURL: &'a str,
    OutputParent: &'a str,
    StorageRoot: &'a str,
    BuildHome: &'a str,
    Runtime: &'a str,
    Tools: &'a str,
}

/// `worker_json` emits the worker configuration record without a final LF.
#[allow(clippy::too_many_arguments)]
pub(super) fn worker_json(
    executable: &str,
    source: &str,
    forgejo_source: &str,
    fountain_release_url: &str,
    output_parent: &str,
    storage_root: &str,
    build_home: &str,
    runtime: &str,
    tools: &str,
) -> String {
    pretty_json(&WorkerRecord {
        Executable: executable,
        Source: source,
        ForgejoSource: forgejo_source,
        FountainReleaseURL: fountain_release_url,
        OutputParent: output_parent,
        StorageRoot: storage_root,
        BuildHome: build_home,
        Runtime: runtime,
        Tools: tools,
    })
}
