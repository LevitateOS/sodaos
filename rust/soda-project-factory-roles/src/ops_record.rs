use soda_json::JsonValue;

pub(crate) use super::records::*;
use crate::emit::{obj, str_value};
use crate::error::{fail, Error};
use crate::fsx;
use crate::ops_inspect;
use crate::proc;
use crate::validate;

pub fn do_start(ctx: &crate::Ctx, data: &JsonValue) -> Result<JsonValue, Error> {
    if !validate::as_object(data).is_some_and(|e| validate::key_set(e, &["op", "id"])) {
        return fail("unsupported start request");
    }
    let pid = validate::check_id(data.get("id").unwrap_or(&JsonValue::Null))?.to_string();
    fsx::ensure_layout(ctx)?;
    let directory = fsx::prep_dir(ctx, &pid)?;
    ops_inspect::refuse_barred(ctx, &directory)?;
    let prestate = start_prestate(ctx, &directory)?;
    let started = match fsx::read_json(ctx, &directory.join("started.json"), 1024) {
        Ok(value) => Some(value),
        Err(Error::Missing) => None,
        Err(err) => return Err(err),
    };
    if let Some(started) = started {
        let pgid = ops_inspect::started_pgid(&started)?;
        if !ops_inspect::group_alive(pgid, std::path::Path::new("/proc")) {
            return fail("preparation supervisor is gone; stop and use a new identity");
        }
        return Ok(obj(vec![
            ("started", str_value(&pid)),
            ("repeated", JsonValue::Bool(true)),
        ]));
    }
    let started = proc::spawn_detached(ctx, &directory, &prestate.fields, &prestate.tools)?;
    let pgid = ops_inspect::started_pgid(&started)?;
    Ok(obj(vec![
        ("started", str_value(&pid)),
        ("repeated", JsonValue::Bool(false)),
        ("pgid", JsonValue::Number(pgid.to_string())),
    ]))
}
