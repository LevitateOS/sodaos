//! Native Cockpit account gate, mirroring `tests/installed/cockpit-account.py`.
//!
//! A real PAM account phase: root must succeed and the existing non-root
//! `nobody` must fail with an explicit permission/auth denial. No account
//! is created, no credentials are ever supplied, and nothing here proves
//! authentication or session context. PAM is loaded dynamically at
//! runtime, exactly like the Python owner's `ctypes` use.
//!
//! Failure rendering differs from Python in one documented way: uncaught
//! Python failures print tracebacks, while this port prints one
//! `Cockpit account failed:` line. Success output is byte-identical.

use std::ffi::CString;

/// PAM conversation aborts prompts: `PAM_CONV_ERR`, never supplying
/// credentials to an unexpected prompt.
extern "C" fn no_prompt(
    _count: libc::c_int,
    _messages: *mut *mut libc::c_void,
    _response: *mut *mut libc::c_void,
    _appdata: *mut libc::c_void,
) -> libc::c_int {
    19
}

#[repr(C)]
struct Conversation {
    conv: extern "C" fn(
        libc::c_int,
        *mut *mut libc::c_void,
        *mut *mut libc::c_void,
        *mut libc::c_void,
    ) -> libc::c_int,
    appdata_ptr: *mut libc::c_void,
}

type PamStart = unsafe extern "C" fn(
    *const libc::c_char,
    *const libc::c_char,
    *const Conversation,
    *mut *mut libc::c_void,
) -> libc::c_int;
type PamAcctMgmt = unsafe extern "C" fn(*mut libc::c_void, libc::c_int) -> libc::c_int;
type PamEnd = unsafe extern "C" fn(*mut libc::c_void, libc::c_int) -> libc::c_int;

/// Cockpit failure kind, mirroring the Python exception taxonomy.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CockpitKind {
    /// Failed assertion.
    AssertionError,
    /// Unknown account name.
    KeyError,
}

impl CockpitKind {
    /// Exception name.
    pub fn name(self) -> &'static str {
        match self {
            CockpitKind::AssertionError => "AssertionError",
            CockpitKind::KeyError => "KeyError",
        }
    }
}

/// Cockpit failure with its detail message.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CockpitFailure {
    /// Failure kind.
    pub kind: CockpitKind,
    /// Detail message (may be empty for bare assertions).
    pub detail: String,
}

impl CockpitFailure {
    fn assertion(detail: impl Into<String>) -> CockpitFailure {
        CockpitFailure {
            kind: CockpitKind::AssertionError,
            detail: detail.into(),
        }
    }
}

impl std::fmt::Display for CockpitFailure {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "Cockpit account failed: {} {}",
            self.kind.name(),
            self.detail
        )
    }
}

impl std::error::Error for CockpitFailure {}

/// Explicit native root target gate, like the Python owner's entry check.
/// Pure over its inputs so tests cover it without privileges.
pub fn check_gate(uid: u32, validate: Option<&str>, hostname: &str) -> Result<(), String> {
    if uid != 0 || validate != Some(hostname) {
        return Err("explicit native root target required".to_string());
    }
    Ok(())
}

fn account_uid(name: &str) -> Result<u32, CockpitFailure> {
    let user = CString::new(name).map_err(|_| CockpitFailure {
        kind: CockpitKind::KeyError,
        detail: format!("getpwnam(): name not found: '{name}'"),
    })?;
    let mut entry: libc::passwd = unsafe { std::mem::zeroed() };
    let mut buffer = vec![0 as libc::c_char; 4096];
    let mut result: *mut libc::passwd = std::ptr::null_mut();
    let rc = unsafe {
        libc::getpwnam_r(
            user.as_ptr(),
            &mut entry,
            buffer.as_mut_ptr(),
            buffer.len(),
            &mut result,
        )
    };
    if rc != 0 || result.is_null() {
        return Err(CockpitFailure {
            kind: CockpitKind::KeyError,
            detail: format!("getpwnam(): name not found: '{name}'"),
        });
    }
    Ok(unsafe { *result }.pw_uid)
}

struct Pam {
    start: PamStart,
    acct_mgmt: PamAcctMgmt,
    end: PamEnd,
}

impl Pam {
    /// `dlopen` libpam and resolve the account-phase symbols.
    unsafe fn load() -> Result<(Pam, *mut libc::c_void), CockpitFailure> {
        let name = CString::new("libpam.so.0").unwrap();
        let library = libc::dlopen(name.as_ptr(), libc::RTLD_NOW);
        if library.is_null() {
            return Err(CockpitFailure::assertion(
                "PAM initialization failed; this is not access denial",
            ));
        }
        let symbol = |symbol_name: &str| {
            let raw = CString::new(symbol_name).unwrap();
            libc::dlsym(library, raw.as_ptr())
        };
        let start_raw = symbol("pam_start");
        let acct_raw = symbol("pam_acct_mgmt");
        let end_raw = symbol("pam_end");
        if start_raw.is_null() || acct_raw.is_null() || end_raw.is_null() {
            libc::dlclose(library);
            return Err(CockpitFailure::assertion(
                "PAM initialization failed; this is not access denial",
            ));
        }
        Ok((
            Pam {
                start: std::mem::transmute::<
                    *mut libc::c_void,
                    unsafe extern "C" fn(
                        *const i8,
                        *const i8,
                        *const Conversation,
                        *mut *mut libc::c_void,
                    ) -> i32,
                >(start_raw),
                acct_mgmt: std::mem::transmute::<
                    *mut libc::c_void,
                    unsafe extern "C" fn(*mut libc::c_void, i32) -> i32,
                >(acct_raw),
                end: std::mem::transmute::<
                    *mut libc::c_void,
                    unsafe extern "C" fn(*mut libc::c_void, i32) -> i32,
                >(end_raw),
            },
            library,
        ))
    }
}

/// Run the native Cockpit account phase for one username.
fn account_phase(pam: &Pam, username: &str, allowed: bool) -> Result<(), CockpitFailure> {
    let service = CString::new("cockpit").unwrap();
    let user = CString::new(username).unwrap();
    let conv = Conversation {
        conv: no_prompt,
        appdata_ptr: std::ptr::null_mut(),
    };
    let mut handle: *mut libc::c_void = std::ptr::null_mut();
    let code = unsafe { (pam.start)(service.as_ptr(), user.as_ptr(), &conv, &mut handle) };
    if code != 0 {
        return Err(CockpitFailure::assertion(
            "PAM initialization failed; this is not access denial",
        ));
    }
    let code = unsafe { (pam.acct_mgmt)(handle, 0) };
    let verdict = if allowed {
        if code != 0 {
            Err(CockpitFailure::assertion(
                "root Cockpit account phase did not succeed",
            ))
        } else {
            Ok(())
        }
    } else if code == 6 || code == 7 {
        Ok(())
    } else {
        Err(CockpitFailure::assertion(
            "expected explicit PAM permission/auth denial, not lookup/transport/expiry failure",
        ))
    };
    let end_code = unsafe { (pam.end)(handle, code) };
    if end_code != 0 {
        return Err(CockpitFailure::assertion(""));
    }
    verdict
}

/// Run the full Cockpit account probe: gate, account existence, then the
/// root/nobody PAM account phases. Returns the success line on completion.
pub fn run_cockpit(
    uid: u32,
    validate: Option<&str>,
    hostname: &str,
) -> Result<String, CockpitFailure> {
    check_gate(uid, validate, hostname).map_err(|detail| CockpitFailure {
        kind: CockpitKind::AssertionError,
        detail,
    })?;
    // No account is created. nobody must actually exist, not merely be unknown.
    if account_uid("root")? != 0 {
        return Err(CockpitFailure::assertion(""));
    }
    if account_uid("nobody")? == 0 {
        return Err(CockpitFailure::assertion(""));
    }
    let (pam, library) = unsafe { Pam::load() }?;
    let outcome = account_phase(&pam, "root", true).and(account_phase(&pam, "nobody", false));
    unsafe {
        libc::dlclose(library);
    }
    outcome?;
    Ok("Native Cockpit account phase admits root and denies existing non-root nobody. No password authentication/session claim.".to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn gate_requires_root_and_matching_validation() {
        assert!(check_gate(0, Some("host"), "host").is_ok());
        assert_eq!(
            check_gate(1000, Some("host"), "host").unwrap_err(),
            "explicit native root target required"
        );
        assert_eq!(
            check_gate(0, Some("other"), "host").unwrap_err(),
            "explicit native root target required"
        );
        assert_eq!(
            check_gate(0, None, "host").unwrap_err(),
            "explicit native root target required"
        );
    }

    #[test]
    fn root_account_resolves_to_uid_zero() {
        assert_eq!(account_uid("root").unwrap(), 0);
        let missing = account_uid("definitely-no-such-user-xyz").unwrap_err();
        assert_eq!(missing.kind, CockpitKind::KeyError);
    }

    #[test]
    fn pam_phases_need_a_native_root_target() {
        // Unprivileged callers fail the gate before touching PAM.
        let err = run_cockpit(1000, Some("host"), "host").unwrap_err();
        assert_eq!(err.detail, "explicit native root target required");
    }
}
