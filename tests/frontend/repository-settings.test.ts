import assert from 'node:assert/strict';
import test from 'node:test';
import {creationProfile} from '../../frontend/spaces/sodaspaces-project-response';

const profile = {
  id: 'rocky-headless',
  distribution: 'rocky',
  version: '10.2',
  interface: 'headless',
  architecture: 'amd64',
  image: 'sha256:' + 'a'.repeat(64),
  revision: 'b'.repeat(40),
};
test('creation identity rejects unsupported and incomplete metadata', () => {
  assert.deepEqual(creationProfile(profile), profile);
  for (const bad of [
    {...profile, id: 'fedora-kde'},
    {...profile, revision: ''},
    {...profile, image: 'latest'},
    {...profile, architecture: 'x86'},
    {...profile, version: '<script>'},
  ])
    assert.throws(() => creationProfile(bad));
});
