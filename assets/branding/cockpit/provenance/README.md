# Retired custom Cockpit provenance

These inputs and license texts are retained from the removed custom Cockpit
workspace. They are not compiled, staged or used by the stock Cockpit branding.
The active adapter is in the parent directory; its README links the owning guide.

The TypeScript and Sass files in this directory are unmodified upstream files
from cockpit-project/cockpit revision
`e2cc3d540107a50d66cf8970037083cfbaba16c2`, under `pkg/lib/`.
They retain their upstream copyright and LGPL-2.1-or-later notices; the license
text is included. Updates must select and review an explicit upstream revision.

Only the dark-theme module and the PatternFly 6 adaptation's two Sass inputs are
included. The former adaptation's Red Hat font references used its then-locked
PatternFly package. No Cockpit client or native adapter is vendored.

Additional license provenance:

- PatternFly MIT: npm 6.6.1 gitHead `26b709bfeb14c3643a6b999a2619b6bd65641ffa`, `LICENSE.txt`.
- PatternFly React MIT: npm 6.6.1 gitHead `a9477a0faacc1cd89ac4e65cec6b8b806c7ba2b3`, `LICENSE`.
- Red Hat fonts OFL: RedHatOfficial/RedHatFont tag `4.0.0`, `LICENSE`.

The retired build collected other runtime package licenses from its locked npm
tarballs. Historical generated bundles retain their own notices; these files do
not describe the active canonical fonts or the current stock package closure.
