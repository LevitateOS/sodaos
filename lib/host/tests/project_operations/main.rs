// Oracle + unit tests for the pops JSON adapter (`src/pops.rs`).
// Unit tests drive every op through a scripted fake `Executor` (no live
// containers); oracle tests assert byte-for-byte parity with Go.
// Golden vectors were captured from this branch with `go run ./popsgolden`
// (`json.Marshal` of the domain structs plus stdin-body maps and strictjson
// errors; helper removed after capture):
use soda_host::pops::{
    AccessKeysReq, AccountReq, HoldPreparationReq, InspectPreparationReq, Ops, PrepareCandidateReq,
    PrepareReq, StopPreparationReq,
};
use soda_host::{account, project};
use std::cell::RefCell;
use std::collections::VecDeque;
use std::time::{Duration, Instant};

use crate::common::{ops, Mock};

mod access_keys;
mod accounts;
mod candidate;
mod common;
mod hold;
mod preparation;
mod requests;

#[test]
fn ops_constructor_mirrors_runtime_fields() {
    let mock = Mock::new(vec![]);
    let o = ops(&mock);
    assert_eq!(o.config.network, "sodanet");
    assert_eq!(o.config.subnet, "10.0.0.0/24");
}
