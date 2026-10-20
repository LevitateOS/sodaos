# Run Project development services

Project services are development databases and application containers inside the shared Project. A Project administrator manages persistent workloads; members connect only to assigned endpoints and data.

## Inspect and start a service

Work in a Project terminal as an administrator authorized to manage its nested Podman engine or systemd services. Keep Compose files and unit definitions with the Project or repository, and coordinate port use with other users. The Project engine is separate from the appliance engine.

From the directory containing `compose.yml`, inspect services and recent logs before explicitly starting one. Replace `SERVICE` with a service name defined in that Compose file:

```sh
podman compose -f compose.yml ps
podman compose -f compose.yml logs --tail=100 SERVICE
podman compose -f compose.yml up -d SERVICE
```

For a Project systemd unit, inspect its status and logs before starting it. Replace `SERVICE` with the unit name defined for the Project:

```sh
systemctl status SERVICE.service
journalctl -u SERVICE.service --no-pager -n 100
systemctl start SERVICE.service
```

After a successful start, the Compose service appears in `ps` or the systemd unit reports active; inspect logs and verify the endpoint before relying on the service. These commands manage development workloads inside the Project and do not deploy production applications.

## Example: a PostgreSQL development service

This example uses the database image, persistent volume and native secret pattern from Soda's maintained Project workload example. It adds a Project-local listener and separate human, coder and reviewer databases. The Project administrator owns the service; ordinary accounts use PostgreSQL without access to the container engine.

### Create the service and its private secret

Reserve the Compose project name `handbook-db` and Project loopback port 55432. Use different names and a free port if these are already assigned. Create and enter a new administrator-controlled service directory:

```sh
install -d -m 0700 "$HOME/services/handbook-db"
cd "$HOME/services/handbook-db"
```

Keep the reviewed definition there as `compose.yaml`:

```yaml
services:
  database:
    image: docker.io/library/postgres:17@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675
    environment:
      POSTGRES_DB: soda_example
      POSTGRES_USER: developer
      POSTGRES_PASSWORD_FILE: /run/secrets/handbook-db-admin
    secrets:
      - source: handbook-db-admin
        uid: "999"
        gid: "999"
        mode: "0400"
    ports:
      - "127.0.0.1:55432:5432"
    volumes:
      - database:/var/lib/postgresql/data
volumes:
  database:
secrets:
  handbook-db-admin:
    external: true
```

The named `database` volume holds PostgreSQL data independently of a terminal. Use the same Compose project name on every command so the same volume is selected. The image initializes the `developer` database administrator only when its data directory is empty; changing the secret file does not reset an existing database password.

Create a private credential directory in your own home:

```sh
install -d -m 0700 "$HOME/.config/handbook-db"
```

Use a private editor to create `$HOME/.config/handbook-db/admin-password` containing a unique password on one line. From the service directory, restrict that file and create the native secret before starting the service:

```sh
chmod 0600 "$HOME/.config/handbook-db/admin-password"
podman secret create handbook-db-admin "$HOME/.config/handbook-db/admin-password"
podman compose -p handbook-db -f compose.yaml up -d database
podman compose -p handbook-db -f compose.yaml ps
podman compose -p handbook-db -f compose.yaml exec -T database \
  pg_isready -U developer -d soda_example
```

Readiness reports “accepting connections”. If initialization is still running, inspect `podman compose -p handbook-db -f compose.yaml logs --tail=100 database` and repeat the readiness read after it finishes. A successful `pg_isready` result checks the server, not an account's password or database permissions. Do not recreate a secret or volume to clear an uncertain startup result.

Keep the administrator password away from ordinary members and Factory roles. [Podman secrets](https://docs.podman.io/en/latest/markdown/podman-secret-create.1.html) provide the file consumed by the [PostgreSQL image](https://hub.docker.com/_/postgres); this example does not use passwords in Compose variables or command arguments.

### Allocate one database to each account

The following names belong to this fresh example:

| Account | Database | Data owner and intended use |
| --- | --- | --- |
| `handbook_member` | `handbook_member` | A human member's retained development data. |
| `handbook_coder` | `handbook_coder` | Assigned disposable coding/test data. |
| `handbook_reviewer` | `handbook_reviewer` | Separate disposable review/test data. |

As the service administrator, open an interactive SQL session inside the database container:

```sh
podman compose -p handbook-db -f compose.yaml exec database \
  psql -X -v ON_ERROR_STOP=1 -U developer -d soda_example
```

Run this SQL once in the new service. Stop and inspect any error instead of pasting it again over existing accounts or databases:

```sql
CREATE ROLE handbook_member LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;
CREATE ROLE handbook_coder LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;
CREATE ROLE handbook_reviewer LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;

CREATE DATABASE handbook_member OWNER handbook_member;
CREATE DATABASE handbook_coder OWNER handbook_coder;
CREATE DATABASE handbook_reviewer OWNER handbook_reviewer;

REVOKE ALL ON DATABASE soda_example, postgres, template1 FROM PUBLIC;
REVOKE ALL ON DATABASE handbook_member, handbook_coder, handbook_reviewer FROM PUBLIC;
```

Each account owns its database and can create or remove test objects there. The roles receive no membership in one another and no server-administration privilege. PostgreSQL normally grants database connection privileges to `PUBLIC`, which includes every role; the revocations prevent that default from opening the other databases. Apply this recipe only to the new example service, not an existing shared database cluster.

Still in psql, set a different private password for each account at the hidden prompts, then exit:

```psql
\password handbook_member
\password handbook_coder
\password handbook_reviewer
\q
```

Use [psql's password prompt](https://www.postgresql.org/docs/current/app-psql.html) so plaintext passwords do not enter SQL history. Give each account only its own credential through your team's private delivery path. The [PostgreSQL privilege rules](https://www.postgresql.org/docs/current/sql-grant.html) govern access; the names alone do not provide isolation.

### Connect as a member and verify a retained write

Have the Project administrator supply the native PostgreSQL client if `psql` is absent; ordinary membership does not grant package installation. In the member's own Project terminal, connect through the loopback port with the member's password:

```sh
psql -X -v ON_ERROR_STOP=1 -h 127.0.0.1 -p 55432 \
  -U handbook_member -d handbook_member -W
```

Check the selected identity and create one small record:

```sql
SELECT current_user, current_database();
CREATE TABLE handbook_note (id integer PRIMARY KEY, message text NOT NULL);
INSERT INTO handbook_note VALUES (1, 'Retain this development note');
SELECT id, message FROM handbook_note;
```

The identity query returns `handbook_member` twice. The final query returns row 1 and its message. Exit with `\q`; later checks select the existing row rather than rerunning `CREATE TABLE` or `INSERT`.

For access from your own computer, forward through your joined Project account. Replace `USER@PROJECT_ADDRESS` with the Project's SSH details and leave this tunnel open:

```sh
ssh -N -L 127.0.0.1:55432:127.0.0.1:55432 USER@PROJECT_ADDRESS
```

Run the same `psql` command locally with your PostgreSQL client. If local port 55432 is occupied, choose another local port in the tunnel and client command; the Project destination remains 55432. This loopback-only example does not expose PostgreSQL on the appliance address.

### Verify the assigned Factory data boundary

Assign the coder and reviewer their own database names and credentials. For noninteractive checks, provide each role a private PostgreSQL password file and set `PGPASSFILE` to its absolute path in that role's preparation environment. The file must be accessible only to that role, normally mode 0600. Follow the [password-file format](https://www.postgresql.org/docs/current/libpq-pgpass.html), using the exact host, port, database and user rather than broad wildcard entries. Keep secret values out of accepted source, prompts, results and shared files.

Run this check as the coder, with its password file already supplied:

```sh
PGCONNECT_TIMEOUT=5 psql -X -w -v ON_ERROR_STOP=1 \
  -h 127.0.0.1 -p 55432 -U handbook_coder -d handbook_coder <<'SQL'
SELECT current_user, current_database();
BEGIN;
CREATE TABLE handbook_readiness_probe (id integer PRIMARY KEY);
INSERT INTO handbook_readiness_probe VALUES (1);
SELECT count(*) FROM handbook_readiness_probe;
ROLLBACK;
DO $$
BEGIN
  IF current_user <> current_database() THEN
    RAISE EXCEPTION 'Connected to the wrong assigned database';
  END IF;
  IF has_database_privilege(current_user, 'handbook_member', 'CONNECT')
     OR has_database_privilege(current_user, 'handbook_reviewer', 'CONNECT') THEN
    RAISE EXCEPTION 'Unexpected access to another assigned database';
  END IF;
END
$$;
SQL
```

Expect the coder's identity, a count of 1 and a successful `DO` result. The transaction leaves no probe table behind. Wrong database selection or access to either other database raises an error and fails the command. Repeat from the reviewer's assigned account, substituting `handbook_reviewer` for the connection user/database and `handbook_coder` for the second denied database. Confirm that the member account is also denied connection to both Factory databases.

These checks establish connectivity, required writes and the database connection boundary. A role must never receive the `developer` credential or the shared engine socket. Keep destructive tests and migrations within the assigned disposable database, and coordinate test resets so they cannot collide with another active use. Record the actual endpoint and non-secret resource identities through [Project preparation](../40-Configure/30-prepare-project.md).

### Restart and check the same data

Coordinate an interruption with members and hold affected Factory work before restarting the service. From the administrator's service directory:

```sh
podman compose -p handbook-db -f compose.yaml stop database
podman compose -p handbook-db -f compose.yaml start database
podman compose -p handbook-db -f compose.yaml exec -T database \
  pg_isready -U developer -d soda_example
```

Wait for readiness, then reconnect as `handbook_member` and select `id, message` from `handbook_note`. Row 1 must still contain “Retain this development note”. Recheck coder and reviewer access from their actual accounts before resuming Factory work.

Project Stop also interrupts the nested service. After Project Start, inspect its state and explicitly start `database` if needed, using the same directory and Compose project name. This example declares no automatic restart policy.

When finished, `stop database` leaves the data and service definition intact. Do not use `down -v`, volume removal or a database reset as terminal cleanup. After all assigned tests have stopped, an account can remove only its own explicitly disposable objects; keep the member's note and other human data until its owner requests removal.

## Connect to assigned endpoints

Members use only the service endpoints and credentials the administrator has made available. Bind listeners to the intended Project interface and coordinate address and port selection. Use an SSH local forward when a service should be reached through SSH instead of exposed on a shared interface. The placeholders below are the local client port, Project-local listener port, and Project login/address shown in access details:

```sh
ssh -L LOCAL_PORT:127.0.0.1:PROJECT_PORT USER@PROJECT_ADDRESS
```

Factory coder and reviewer roles also need explicit endpoint access, narrowly scoped credentials, and disposable test data. A prompt or readiness claim does not grant access by itself. Test service health and required permissions from the assigned account. Never run destructive tests or migrations against shared human data; changes to shared data require the appropriate Project authority.

## Coordinate service ownership

Record which account owns each development service and who may stop it. Native Project root and access to the shared Podman engine are trusted administrative privileges; membership alone does not grant them. Prefer an independently administered service identity for a deliberately shared workload rather than an incidental process in a person's terminal.

Ending a terminal stops its supervised processes; a server started inside that terminal can stop too. An independent systemd or container service has its own lifetime. Project Stop interrupts both. Revoking a member ends processes under that member's identity, including services they own; review those affected workloads before offboarding. Keep shared data and service ownership explicit so revocation preserves independently owned team services.

## Preserve data and recheck readiness

Keep durable data in the service's intended Project volume or persistent path. Terminal End does not stop independent services or delete their volumes. Project Stop interrupts users and stops live workloads while preserving Project data.

After a restart or service replacement, check status and logs, then verify reachability and required permissions from the actual assigned account. Reconnecting a terminal does not establish service readiness. Factory resource assignment belongs to [Project preparation](../40-Configure/30-prepare-project.md).

For a Canlang application preview, use [Canlang development](70-canlang-development.md). An independently installed private application has its own lifecycle in [Private applications](../50-Operate/40-private-applications.md).
