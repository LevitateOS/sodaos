"""Candidate-gate wiring: check-native.sh must use the Go validator owner.

The gate binds the requested architecture and Soda/Fountain revisions and
verifies the delivered archives through Candidate.Validate and
VerifyCandidateImages. It keeps no independent Python candidate rules.
"""

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
NATIVE = ROOT / 'scripts/check-native.sh'

# Distinctive messages of the retired independent Python candidate rules.
RETIRED_RULES = [
    'candidate architecture mismatch',
    'candidate provenance architecture mismatch',
    'independent extension image identity required',
    'pinned Forgejo compiler provenance required',
    'resolved Forgejo APK provenance required',
    'host and candidate content inventories differ',
    'extension installer CLI differs from patched Forgejo',
    'unsafe extension asset inventory',
    'incomplete candidate content inventory',
    'candidate content hash mismatch: ',
    'host package inventory hash mismatch',
    'missing application archives: ',
]


class CandidateGateWiring(unittest.TestCase):
    def test_candidate_check_uses_the_go_validator_owner(self):
        native = NATIVE.read_text()
        before, after = native.split('bun run check:source\n')
        self.assertIn('tools/soda-candidate-check', before)
        self.assertIn('--candidate "$artifacts"', before)
        self.assertIn('--arch "$arch"', before)
        self.assertIn('--soda-revision "$revision"', before)
        self.assertIn('--forgejo-revision "$forgejo_revision"', before)
        self.assertIn('git -C ../forgejo-ext rev-parse HEAD', before)
        self.assertIn('Exact Fountain source revision required', before)
        self.assertNotIn('soda-candidate-check', after)

    def test_no_independent_python_candidate_rules(self):
        native = NATIVE.read_text()
        self.assertNotIn('python3 - <<PY', native)
        for rule in RETIRED_RULES:
            with self.subTest(rule=rule):
                self.assertNotIn(rule, native)

    def test_native_guards_still_surround_the_owner_check(self):
        native = NATIVE.read_text()
        before, after = native.split('bun run check:source\n')
        self.assertIn('Pinned native source-check tools required', before)
        self.assertIn('Check requires a clean exact-revision checkout', before)
        self.assertIn(
            'soda-build candidate artifacts directory required (payload.json, candidate.json, host.oci)', before
        )
        self.assertIn('"$verifier" verify --source "$artifacts" --arch "$arch" --revision "$revision"', before)
        self.assertIn('$(git rev-parse HEAD) == "$revision"', after)


if __name__ == '__main__':
    unittest.main()
