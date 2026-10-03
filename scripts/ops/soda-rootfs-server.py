"""Exact-file rootfs server (stdlib only). Serves ONLY ^[0-9a-f]{64}-rootfs\\.img$.

Deny by construction: no directory listing, no QCOW2, no ISO, no traversal,
no symlinks, no subpaths, GET/HEAD only. Anything else is 404/501.

Single source for soda-rootfs-server.service: the unit runs this file, and
scripts/setup-soda-candidate.sh creates the ROOT directory below. There is
no second copy; behavior checks live in tests/build/test_rootfs_serving.py.
"""

import http.server
import os
import re
import stat
import urllib.parse

ROOT = b"/home/soda-rootfs"
ALLOW = re.compile(rb"^[0-9a-f]{64}-rootfs\.img$")
CHUNK = 1048576


class Handler(http.server.BaseHTTPRequestHandler):
    server_version = "soda-rootfs/1"

    def _refuse(self):
        self.send_error(404)
        return None

    def _open_allowed(self):
        raw = urllib.parse.urlparse(self.path).path
        try:
            name = urllib.parse.unquote(raw, encoding="ascii", errors="strict")
        except (UnicodeDecodeError, ValueError):
            return self._refuse()
        if not name.startswith("/") or "/" in name[1:]:
            return self._refuse()
        token = name[1:].encode("ascii", "strict")
        if not ALLOW.match(token):
            return self._refuse()
        try:
            fd = os.open(os.path.join(ROOT, token), os.O_RDONLY | os.O_NOFOLLOW)
        except OSError:
            return self._refuse()
        try:
            if not stat.S_ISREG(os.fstat(fd).st_mode):
                os.close(fd)
                return self._refuse()
            return os.fdopen(fd, "rb", buffering=0)
        except OSError:
            try:
                os.close(fd)
            except OSError:
                pass
            return self._refuse()

    def _serve(self):
        body = self._open_allowed()
        if body is None:
            return
        try:
            size = os.fstat(body.fileno()).st_size
            self.send_response(200)
            self.send_header("Content-Type", "application/octet-stream")
            self.send_header("Content-Length", str(size))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            if self.command == "GET":
                while True:
                    chunk = body.read(CHUNK)
                    if not chunk:
                        break
                    self.wfile.write(chunk)
        finally:
            body.close()

    def do_GET(self):
        self._serve()

    def do_HEAD(self):
        self._serve()

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    httpd = http.server.ThreadingHTTPServer(("192.168.122.1", 8080), Handler)
    httpd.serve_forever()
