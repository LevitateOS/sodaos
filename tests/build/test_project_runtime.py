"""Source contracts for reproduced native project service/secret failures."""
import configparser
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


class ProjectRuntimeContracts(unittest.TestCase):
    def unit(self, name):
        config = configparser.ConfigParser(interpolation=None)
        config.read(ROOT / 'project-os/rootfs/etc/systemd/system' / name)
        return config

    def test_service_uses_local_engine_and_activation_fd(self):
        unit = self.unit('soda-podman.service')
        service = unit['Service']
        self.assertEqual(service['UnsetEnvironment'], 'CONTAINER_HOST')
        self.assertNotIn('Environment', service)
        self.assertEqual(service['ExecStart'], '/usr/bin/podman system service --time=0')
        self.assertIn('soda-podman.socket', unit['Unit']['Requires'].split())
        # A stopping service must not remove the socket unit's directory.
        self.assertNotIn('RuntimeDirectory', service)

    def test_socket_is_project_admin_only(self):
        unit = self.unit('soda-podman.socket')
        socket = unit['Socket']
        self.assertEqual(socket['ListenStream'], '/run/soda-podman/podman.sock')
        self.assertEqual(socket['SocketUser'], 'root')
        self.assertEqual(socket['SocketGroup'], 'wheel')
        self.assertEqual(socket['SocketMode'], '0660')
        self.assertEqual(socket['DirectoryMode'], '0750')
        self.assertIn('soda-project-init.service', unit['Unit']['After'].split())
        init = (ROOT / 'project-os/rootfs/usr/libexec/soda/project-init').read_text()
        self.assertIn('install -d -m 0750 -o root -g wheel /run/soda-podman', init)
        recipe = (ROOT / 'project-os/Containerfile').read_text()
        self.assertIn('systemctl enable sshd.service soda-project-init.service soda-podman.socket', recipe)

    def test_compose_uses_native_secret_not_password_argv(self):
        compose = (ROOT / 'tests/fixtures/workload/compose.yaml').read_text()
        self.assertIn('POSTGRES_PASSWORD_FILE: /run/secrets/soda-example-db', compose)
        self.assertNotIn('POSTGRES_PASSWORD:', compose)
        self.assertNotIn('${EXAMPLE_DB_PASSWORD', compose)
        self.assertIn('external: true', compose)
        self.assertIn('mode: "0400"', compose)


if __name__ == '__main__':
    unittest.main()
