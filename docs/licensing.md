# Licensing and retained source-review notes

Original SodaOS code/documentation/configuration are covered by the root
[Apache-2.0 LICENSE](../LICENSE), with scope and attribution in [NOTICE](../NOTICE)
and the README. This grant does not relicense inherited code, canonical artwork
or third-party material. Preserve their existing terms and provenance.

These notes preserve useful findings from the removed Forgejo source-build guide.
They are not a build recipe, dependency lock, distribution approval or complete
license-clearance claim. The selected integration uses stock Forgejo and supported
template overrides; no patched executable or native source-build path is selected.

## Distribution obligations

- Forgejo's overall GPL-3.0-or-later and its per-file/template terms remain applicable.
  Swagger's MIT grant is not the Forgejo distribution license. Using an official
  image or overriding a template does not waive corresponding-source/notice duties.
- Preserve copyright/license notices in overridden upstream templates and assets,
  document modifications as required and deliver their applicable source/notices.
  Do not stamp inherited templates as original Apache-only Soda code.
- U02's actual bundle must bind the stock image, overrides and Soda artifacts to
  their source/notices. Include required source and licenses for distributed
  dependencies/runtime material; a tag URL or generic license-name list is not
  automatically sufficient. Exclude private configuration, credentials, databases,
  repositories and evidence from distributable payloads.
- Retain MIT/BSD notices, Apache NOTICE, applicable MPL covered source, LGPL
  source/relinking requirements and OFL/CC attribution according to actual shipped
  content. Review embedded assets/linkage, not only container labels or `dev` flags.
- The inspected predecessor checkout had no root LICENSE/COPYING grant. Preserve
  file-level/vendor rights and [provenance](predecessor-reuse.md); do not infer that
  inherited code/artwork is Apache-licensed merely because it resides here.
  Unresolved rights/source obligations stop distribution; they are not fixed by
  deleting a notice or assuming permission. The predecessor remains unchanged.

## Retained concrete findings

From the U01 review at `542de21`, whose architecture acceptance was later withdrawn:

- Apache license text was retrieved verbatim from the Apache Software Foundation.
- PatternFly 6.6.1's actual `LICENSE.txt` at published npm Git commit
  `26b709bfeb14c3643a6b999a2619b6bd65641ffa` grants MIT, copyright 2019 Red Hat, Inc.
  All 33 installed font files matched that source's Git blob IDs. Those include
  Red Hat Display/Text/Mono and Font Awesome Solid.
- Retrieved Red Hat Font OFL/author texts as verified Git blobs, including Reserved
  Font Name Red Hat. Font Awesome 5.0.13 distinguishes fonts (OFL-1.1), SVG/JS icons
  (CC-BY-4.0) and other code (MIT). Actual shipped-font/notice pairing remains a
  packaging check; an arbitrary current OFL text is not proof for every version.
- The shared `cockpit/build/licenses.ts` collector exempts `@patternfly/*` from
  missing-license-text refusal; dashboard uses it too. U02 must supply actual CSS/
  emitted font/icon notices, not rely on this exemption as complete compliance.
- Top-level license/NOTICE texts for all 13 modules named in Soda's go.mod were
  reviewed from exact-version caches: MIT/BSD and YAML's per-file MIT/Apache split
  plus NOTICE. This is not the full dependency graph or final binary closure.
- The researched Forgejo v16.0.3 npm lock had 1,220 entries. Three missing license
  declarations (`khroma` 2.1.0, `reserved` 0.1.2, `svg-tags` 1.0.0) were resolved
  from exact integrity-verified tarballs as MIT. That research does not select v16
  for deployment or certify the shipped stock image's closure.
- The researched upstream Go-license target tolerated collection errors, and its
  generator omitted NOTICE/README. Its 260 generated notices lacked version/build-
  tag binding. Do not present that metadata as complete actual-artifact compliance.
- Retain existing Cockpit LGPL, HTMX and Tea license texts, runner payload notices
  and canonical branding attribution; no asset/license removal is implied here.

Research evidence remains in `.artifacts/research/u01-d5b5065/` and
`.artifacts/research/u01-8727233/`, with provenance, license texts, hash comparisons
and retained failures. Earlier source-preparation evidence is historical only.
The [handoff](implementation-status.md) records exactly what ran. No notices have
been newly packaged or installed by moving these notes.

## Upgrade research is not a deployment selection

The prior v15/v16 comparison and release-note review remain evidence in H01 and the
handoff. Stock 15.0.7 is still the recorded installation. The v16 source development
lock was removed with the fork preparer; missing JSON APIs no longer justify an
upgrade or backport. Select a supported stock release based on actual security,
template/configuration/protocol compatibility and migration review at that time.
Only Forgejo performs its migrations; an old image alone is not data rollback.
