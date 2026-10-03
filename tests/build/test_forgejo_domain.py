"""soda-forgejo-domain must report a malformed app.ini cleanly, never traceback."""

import contextlib
import io
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "appliance/bin/soda-forgejo-domain"


def load_domain():
    """Import the extensionless operator script without running its CLI dispatch."""
    source = SCRIPT.read_text()
    head, marker, _ = source.partition("\nparser = argparse.ArgumentParser")
    if not marker:
        raise AssertionError("soda-forgejo-domain CLI dispatch moved; update this loader")
    namespace = {"__name__": "soda_forgejo_domain_under_test"}
    exec(compile(head, str(SCRIPT), "exec"), namespace)
    return namespace


class FakeResult:
    def __init__(self, returncode=0, stdout=""):
        self.returncode = returncode
        self.stdout = stdout


class ForgejoDomainConfig(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.domain = load_domain()

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.app_ini = self.root / "app.ini"
        self.env_file = self.root / "forgejo.env"
        self.data_root = self.root / "data"
        self.outputs = {
            "systemctl-is-active": FakeResult(1, ""),
            "systemctl-is-enabled": FakeResult(0, "enabled\n"),
            "podman": FakeResult(0, ""),
        }
        domain = self.domain
        original_run = domain["run"]
        original_marker_defaults = domain["marker_path"].__defaults__

        def fake_run(*cmd):
            if cmd[:2] == ("systemctl", "is-active"):
                return self.outputs["systemctl-is-active"]
            if cmd[:2] == ("systemctl", "is-enabled"):
                return self.outputs["systemctl-is-enabled"]
            if cmd[0] == "podman":
                return self.outputs["podman"]
            if cmd[:2] == ("systemctl", "start"):
                self.outputs["systemctl-is-active"] = FakeResult(0, "")
                return FakeResult(0, "")
            raise AssertionError("unexpected command %r" % (cmd,))

        domain["run"] = fake_run
        domain["marker_path"].__defaults__ = (self.env_file, self.app_ini, self.data_root)
        self.addCleanup(lambda: domain.__setitem__("run", original_run))
        self.addCleanup(lambda: setattr(domain["marker_path"], "__defaults__", original_marker_defaults))

    def run_verb(self, verb):
        stdout, stderr = io.StringIO(), io.StringIO()
        code = 0
        with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
            try:
                self.domain["cmd_" + verb](None)
            except SystemExit as exited:
                code = exited.code
        return code, stdout.getvalue(), stderr.getvalue()

    def test_bare_app_name_status_reports_clear_error(self):
        self.app_ini.write_text("APP_NAME = Soda\n")
        code, stdout, stderr = self.run_verb("status")
        self.assertEqual(code, 1)
        self.assertIn("marker: unknown", stdout)
        self.assertIn("soda-forgejo-domain:", stderr)
        self.assertIn("app.ini", stderr)
        self.assertNotIn("Traceback", stdout + stderr)

    def test_bare_app_name_start_has_no_crash(self):
        self.app_ini.write_text("APP_NAME = Soda\n")
        code, stdout, stderr = self.run_verb("start")
        self.assertNotIn("Traceback", stdout + stderr)
        self.assertIn("soda-forgejo-domain:", stderr)
        self.assertIn("app.ini", stderr)

    def test_valid_app_ini_still_resolves(self):
        self.app_ini.write_text("[server]\nAPP_DATA_PATH = /data/soda\n")
        resolve = self.domain["app_data_path"]
        self.assertEqual(resolve(self.env_file, self.app_ini), "/data/soda")
        self.env_file.write_text("FORGEJO__server__APP_DATA_PATH=/data/override\n")
        self.assertEqual(resolve(self.env_file, self.app_ini), "/data/override")

    def test_parsed_but_missing_app_data_path_reports_clear_error(self):
        self.app_ini.write_text("[server]\nAPP_NAME = Soda\n")
        with self.assertRaises(SystemExit) as exited:
            with contextlib.redirect_stderr(io.StringIO()):
                self.domain["app_data_path"](self.env_file, self.app_ini)
        self.assertEqual(exited.exception.code, 1)


if __name__ == "__main__":
    unittest.main()
