import test from 'node:test';
import assert from 'node:assert/strict';
import {readFailureText} from '../../frontend/spaces/sodaspaces-factory-work';

test('read failures separate missing from unavailable', () => {
  assert.deepEqual(readFailureText(404, {}, 'Item not found.', 'Item unavailable.'), {
    missing: true,
    text: 'Item not found.',
  });
  assert.deepEqual(readFailureText(403, {}, 'Item not found.', 'Item unavailable.'), {
    missing: false,
    text: 'Access changed. Sign in again or ask for access.',
  });
  assert.deepEqual(readFailureText(503, {message: 'Store is down.'}, 'Item not found.', 'Item unavailable.'), {
    missing: false,
    text: 'Store is down.',
  });
  assert.deepEqual(readFailureText(0, {}, 'Item not found.', 'Item unavailable.'), {
    missing: false,
    text: 'Item unavailable.',
  });
});
