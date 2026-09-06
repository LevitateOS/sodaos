#!/usr/bin/env python3
"""P06: real PAM account gate, not authentication or session-context proof."""
import ctypes
import ctypes.util
import os
import platform
import pwd

if os.getuid() != 0 or os.environ.get('SODA_NATIVE_VALIDATE') != platform.node():
    raise SystemExit('explicit native root target required')
# No account is created. nobody must actually exist, not merely be unknown.
assert pwd.getpwnam('root').pw_uid == 0
assert pwd.getpwnam('nobody').pw_uid != 0
pam = ctypes.CDLL(ctypes.util.find_library('pam') or 'libpam.so.0')
CONV = ctypes.CFUNCTYPE(ctypes.c_int, ctypes.c_int, ctypes.c_void_p, ctypes.c_void_p, ctypes.c_void_p)


@CONV
def no_prompt(count, messages, response, appdata):
    return 19  # PAM_CONV_ERR; never provide credentials to an unexpected prompt


class Conversation(ctypes.Structure):
    _fields_ = [('conv', CONV), ('appdata_ptr', ctypes.c_void_p)]


pam.pam_start.argtypes = [ctypes.c_char_p, ctypes.c_char_p, ctypes.POINTER(Conversation), ctypes.POINTER(ctypes.c_void_p)]
pam.pam_acct_mgmt.argtypes = [ctypes.c_void_p, ctypes.c_int]
pam.pam_end.argtypes = [ctypes.c_void_p, ctypes.c_int]
for username, allowed in [('root', True), ('nobody', False)]:
    handle = ctypes.c_void_p()
    conv = Conversation(no_prompt, None)
    code = pam.pam_start(b'cockpit', username.encode(), ctypes.byref(conv), ctypes.byref(handle))
    assert code == 0, 'PAM initialization failed; this is not access denial'
    try:
        code = pam.pam_acct_mgmt(handle, 0)
        if allowed:
            assert code == 0, 'root Cockpit account phase did not succeed'
        else:
            assert code in (6, 7), 'expected explicit PAM permission/auth denial, not lookup/transport/expiry failure'
    finally:
        assert pam.pam_end(handle, code) == 0
print('Native Cockpit account phase admits root and denies existing non-root nobody. No password authentication/session claim.')
