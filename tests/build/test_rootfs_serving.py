"""D3: setup, build filing, and the persistent server share one rootfs-only directory.

The installer ISO is only the boot menu; the guest fetches the hash-named
rootfs image over HTTP after the builder exits. These checks read the real
setup script, the staged systemd unit, and the single server source: the
three must agree on one directory that holds only rootfs images, served
without listings, QCOW2, ISO, traversal, subpaths, symlinks, or non-GET/HEAD
methods. No services, units, or host paths are touched.
"""

import http.client
import importlib.util
import os
import re
import socket
import threading
from http.server import ThreadingHTTPServer
from pathlib import Path
import shutil
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SETUP = ROOT / "scripts/setup-soda-candidate.sh"
UNIT = ROOT / "scripts/ops/soda-rootfs-server.service"
SERVER = ROOT / "scripts/ops/soda-rootfs-server.py"

# Live guest-disk directory: QCOW2s and installer ISOs live here and must
# never be a served document root again.
GUEST_IMAGES = "/home/libvirt/images"

ALLOW_NAME = "f" * 64 + "-rootfs.img"


def setup_rootfs_dir(text):
    match = re.search(r'^ROOTFS_DIR="([^"]+)"', text, re.M)
    if not match:
        raise AssertionError("setup script names no ROOTFS_DIR")
    return match.group(1)


def server_root(path):
    match = re.search(r'^ROOT\s*=\s*b"([^"]+)"', path.read_text(), re.M)
    if not match:
        raise AssertionError(f"{path} names no served ROOT")
    return match.group(1)


def load_server(root):
    spec = importlib.util.spec_from_file_location("soda_rootfs_server", SERVER)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    module.ROOT = os.fsencode(str(root))
    return module


class RootfsServing(unittest.TestCase):
    def test_setup_and_server_converge_on_one_home_directory(self):
        served = server_root(SERVER)
        self.assertEqual(
            setup_rootfs_dir(SETUP.read_text()),
            served,
            "setup and the persistent server must file and serve the same directory",
        )
        self.assertTrue(
            served.startswith("/home/"),
            f"rootfs images are gigabytes; {served} must stay off the small root filesystem",
        )
        self.assertNotEqual(served, GUEST_IMAGES, "the guest-disk directory must not be served")

    def test_unit_runs_the_single_server_source(self):
        unit = UNIT.read_text()
        self.assertNotIn("b64decode", unit, "the unit must not embed a duplicate encoded server")
        self.assertNotRegex(unit, r"^#S", "the unit must not embed a duplicate commented server")
        installs = [line for line in unit.splitlines() if line.startswith("ExecStart=")]
        self.assertEqual(len(installs), 1, "the unit must run exactly one server command")
        self.assertIn(
            SERVER.name,
            installs[0],
            "ExecStart must run the single server source from scripts/ops",
        )
        self.assertIn("192.168.122.1", installs[0] + "\n" + unit, "guests keep fetching the same bridge address")

    def test_exact_file_serving(self):
        work = Path(tempfile.mkdtemp(prefix="soda-rootfs-"))
        self.addCleanup(shutil.rmtree, work, True)
        (work / ALLOW_NAME).write_bytes(b"installer-rootfs-bytes")
        (work / "soda-iso29.qcow2").write_bytes(b"guest-disk")
        (work / "soda-iso29-installer.iso").write_bytes(b"installer-iso")
        (work / "notes.txt").write_bytes(b"unrelated")
        (work / "sub").mkdir()
        (work / "sub" / ALLOW_NAME).write_bytes(b"nested")
        try:
            os.symlink(ALLOW_NAME, work / ("e" * 64 + "-rootfs.img"))
        except OSError:
            self.skipTest("symlinks unavailable")
        module = load_server(work)
        server = ThreadingHTTPServer(("127.0.0.1", 0), module.Handler)
        port = server.server_address[1]
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(server.server_close)
        self.addCleanup(thread.join)
        self.addCleanup(server.shutdown)

        def fetch(method, path):
            conn = http.client.HTTPConnection("127.0.0.1", port, timeout=10)
            try:
                conn.request(method, path)
                response = conn.getresponse()
                return response.status, response.getheader("Cache-Control"), response.read()
            finally:
                conn.close()

        status, cache, body = fetch("GET", "/" + ALLOW_NAME)
        self.assertEqual(status, 200, "the exact installer rootfs must be fetchable")
        self.assertEqual(body, b"installer-rootfs-bytes")
        self.assertEqual(cache, "no-store")
        status, _, body = fetch("HEAD", "/" + ALLOW_NAME)
        self.assertEqual(status, 200)
        self.assertEqual(body, b"")
        for path in (
            "/",
            "/soda-iso29.qcow2",
            "/soda-iso29-installer.iso",
            "/notes.txt",
            "/sub/" + ALLOW_NAME,
            "/%2e%2e/etc/passwd",
            "/" + "e" * 64 + "-rootfs.img",
        ):
            with self.subTest(path=path):
                status, _, _ = fetch("GET", path)
                self.assertEqual(status, 404)
        with socket.create_connection(("127.0.0.1", port), timeout=10) as raw:
            raw.sendall(b"POST /" + ALLOW_NAME.encode() + b" HTTP/1.0\r\nContent-Length: 0\r\n\r\n")
            reply = raw.recv(64).decode("latin1")
        self.assertIn("501", reply.splitlines()[0], "only GET/HEAD are served")


if __name__ == "__main__":
    unittest.main()
