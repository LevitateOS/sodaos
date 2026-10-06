use super::*;
use crate::project::PROJECT_INSPECT_FORMAT;
use std::cell::RefCell;

struct FakeExec {
    out: Vec<u8>,
    err: Option<String>,
    seen: RefCell<Vec<String>>,
}

impl Executor for FakeExec {
    fn run(
        &self,
        _stdin: &[u8],
        cmd: &str,
        args: &[&str],
        _deadline: Instant,
    ) -> Result<Vec<u8>, String> {
        let mut seen = self.seen.borrow_mut();
        seen.push(cmd.to_string());
        seen.extend(args.iter().map(|s| s.to_string()));
        match &self.err {
            Some(e) => Err(e.clone()),
            None => Ok(self.out.clone()),
        }
    }
}

fn inspection_json(id: &str, cid: &str, running: bool, owner: &str, privileged: bool) -> Vec<u8> {
    format!(
        "{{\"id\":{cid:?},\"running\":{running},\"project\":{id:?},\"owner\":{owner:?},\"privileged\":{privileged},\"userns\":\"private\",\"mappings\":{{\"UidMap\":[\"0:524288:262144\"],\"GidMap\":[\"0:524288:262144\"]}}}}"
    )
    .into_bytes()
}

#[test]
fn inspect_project_argv_and_gates() {
    let id = "p0123456789abcdef01234567";
    let cid = "b".repeat(64);
    let deadline = Instant::now() + std::time::Duration::from_secs(5);
    let exec = FakeExec {
        out: inspection_json(id, &cid, true, "1000", false),
        err: None,
        seen: RefCell::new(Vec::new()),
    };
    let v = inspect_project(&exec, id, deadline).unwrap();
    assert_eq!(v.id, cid);
    assert!(v.running);
    assert_eq!(project_container(&exec, id, true, deadline).unwrap(), cid);
    assert!(project_running(&exec, id, deadline).unwrap());
    let seen: Vec<String> = exec.seen.borrow().clone();
    let expected = vec![
        "/usr/bin/podman".to_string(),
        "--remote=false".to_string(),
        "inspect".to_string(),
        "--format".to_string(),
        PROJECT_INSPECT_FORMAT.to_string(),
        format!("soda-{id}"),
    ];
    assert_eq!(seen[..6].to_vec(), expected);
    assert_eq!(
        inspect_project(&exec, "bogus", deadline).unwrap_err(),
        "invalid project"
    );
    let exec = FakeExec {
        out: inspection_json(id, &cid, false, "1000", false),
        err: None,
        seen: RefCell::new(Vec::new()),
    };
    assert_eq!(
        project_container(&exec, id, true, deadline).unwrap_err(),
        "terminal target not ready or isolated"
    );
    assert_eq!(project_container(&exec, id, false, deadline).unwrap(), cid);
    for (owner, privileged) in [("0", false), ("-3", false), ("NaN", false), ("1000", true)] {
        let exec = FakeExec {
            out: inspection_json(id, &cid, true, owner, privileged),
            err: None,
            seen: RefCell::new(Vec::new()),
        };
        assert_eq!(
            inspect_project(&exec, id, deadline).unwrap_err(),
            "terminal target not ready or isolated",
            "{owner}/{privileged}"
        );
    }
    let exec = FakeExec {
        out: b"{}".to_vec(),
        err: Some("boom".to_string()),
        seen: RefCell::new(Vec::new()),
    };
    assert_eq!(
        inspect_project(&exec, id, deadline).unwrap_err(),
        "terminal inspection unavailable"
    );
    let exec = FakeExec {
        out: b"[]".to_vec(),
        err: None,
        seen: RefCell::new(Vec::new()),
    };
    assert_eq!(
        inspect_project(&exec, id, deadline).unwrap_err(),
        "invalid terminal inspection"
    );
}
