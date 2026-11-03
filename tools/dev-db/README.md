# Development database copy upgrade

`dev-db` emits SQL for `soda-dev` to apply to its retained `soda_dev` database
copy. It imports the canonical store table definitions and does not connect to
PostgreSQL. The SQL refuses any other database name or unknown schema version.

```sh
go run ./tools/dev-db > /home/OPERATOR/dev-schema.sql
```

Versions 67 and 68 receive their missing application tables. Versions through 69
receive the feed-post conversion, and versions through 70 receive the bounded
human-comment command payload upgrade. Known later versions chain the human edit,
issue creation, lock, review, inline, thread, reviewer-request and merge capture allowances
through schema 79. These run in one transaction; an already-current copy is
unchanged. A future source schema version requires reviewing this
development upgrade. The [Forge development workflow](../../docs/development/forge-development.md)
owns cloning, private connection files and deployment.
