# Forgejo and GitHub CLIs in projects

Project images include packaged `tea` (Forgejo) and `gh` (GitHub) CLIs for personal
provider authentication inside the project. They are not shared credentials and not
a substitute for native Git or a factory execution identity.

Runtime packaging lives in `system/project/` recipes. Versions belong in those recipes
and locks, not a second prose table here.

## Usage

- Authenticate each CLI with the member's own provider login inside the project.
- Do not place provider tokens in `~/shared` or the repository.
- Factory credentials and publication authority belong to the separate
  [operator interface](../reference/factory.md), outside personal project logins.

## Maintenance

New tools appear in newly built images. Existing persistent projects receive required
additions only through [same-root maintenance](../reference/project-os.md#same-root-maintenance),
not recreation or an assumed automatic image upgrade.

Related: [Develop](develop.md), [Project OS](../reference/project-os.md).
