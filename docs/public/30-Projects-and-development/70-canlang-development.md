# Develop Canlang applications in a Project

The Project includes Canlang's compiler, platform tools, runtime, and example testkit. Use them to compile an application, run a private development preview and examples, and connect an editor over standard LSP.

## Start with an application and examples

Work in your own Project account and personal checkout. The installed compiler and platform tools provide the runtime; you supply application source and examples. Save this Todo application as `app.can` in a new development directory, or use your repository's existing source instead:

```canlang
# Keep track of what needs doing.
app Todos
Given
 Todo { title:text trim min=1 max=200 label="Task", done:bool=false label={text="Status",values={true="Done",false="Open"}} } label="Todo"
 policy Todo read=members
 fixture task=Todo {title="Try Can"}
When
 crud Todo by=members fields=title,done label={create="Add task"}
  examples update record=task
   as,changes.done,request.record.version -> record.done
   members,true,1 -> true
   members,true,2 -> error(conflict)
   members,true,0 -> error(validation)
   public,true,1 -> error(forbidden)
Then
 preferences { view:enum(all,open,done)=all label={text="Show",values={all="All",open="Open",done="Done"}} }
 page / title="Todos"
  card "Add a task"
   form Todo.create fields=title display=inline
    input title
  card "Your tasks"
   tabs preferences.view
   list Todo as task where preferences.view==all or (preferences.view==open and not task.done) or (preferences.view==done and task.done) order=-created search=title empty="No tasks here. Add a task or choose another view."
    text row.title,row.done
    edit fields=title,done
    tooltip "Permanently delete this task"
     delete
```

This source defines a Todo type accessible to members, a fixture and four update examples: a valid update, a stale-version conflict, an invalid version and a forbidden public update. The examples are executable expectations, not four production records. Keep the source in your native repository so collaborators and Factory preparation use the same reviewed input.

## Check and compile an application

From the directory containing `app.can`, check its source before compiling:

```sh
can check app.can
can compile --format=json app.can > app.json
```

The check reports source diagnostics; successful compilation writes `app.json`. If either fails, correct the reported source problem before starting a preview. The compiler output alone does not start an application or establish browser behavior.

## Preview and run source examples

Start a private development preview for that same source:

```sh
can dev start --source app.can
can dev check
can dev preview.open
```

Inspect the check result, then open `result.url` from `can dev preview.open` in your browser exactly as returned. Sign in with an account from `result.actors`, using its returned email and password, then select one of that account's returned teams. These are private development accounts, separate from your Forgejo sign-in and production application accounts; keep their credentials and preview URL private.

The Todo preview lets a member create a task, change its status and filter the list. Inspect the reported current revision before running examples.

Replace `REVISION` below with the actual revision from your development session, such as r1 only when that is what the session reports:

```sh
can dev example.run --expected-revision REVISION
```

For the unchanged Todo source above, inspect results for all four examples and confirm no failed or unsupported cases. A report with zero executed cases is not success: example execution refuses an empty suite. A changed revision requires checking the current source and using its reported revision rather than replaying an old value. Examples use disposable databases and do not mutate preview data.

## Connect an editor through LSP

Configure your editor's standard LSP integration to launch:

```sh
can lsp
```

This is a foreground language-server process used by the editor for diagnostics, completion, navigation and formatting. Run it through the editor or a separate terminal; it is not a one-shot step between preview commands. An external editor can reach the Project through [SSH development access](40-connect-and-develop.md).

## Stop the private preview

When finished, run this from a normal shell in the application directory:

```sh
can dev stop
```

The preview stops; retained development data remains available to the same source on restart. Stop editor-owned language-server processes through your editor separately. Neither action stops an independently installed private application.

## Keep development state separate

Compiler and platform tools are installed together. The preview keeps development state under your account, separate from production application data and identity. The ordinary `can dev` preview supplies D1, Identity and supported local bindings, including Files backed by local R2. It rejects scheduled handlers and migrations.

Use the installed `can-soda` target to exercise the selected Soda runtime's native Files storage and durable scheduled work. It uses a separate development data directory, explicit development owner bootstrap and an HTTPS origin or proxy for Files. A successful local preview does not establish the behavior of that native storage provider. Never point development or example execution at production credentials or storage.

Preview state defaults to `~/.local/state/canlang/preview`, keyed by the canonical source. `CAN_DEV_STATE_ROOT` selects a separate private development root.

Canlang refuses source or schema changes that cannot safely use retained development state. Choose a fresh development data root when you intentionally need clean development state; this does not migrate production state.

## Exercise native Files with development data

Keep the Todo preview for quick editing and example runs. For native storage and scheduling, use Canlang's maintained `SodaRuntime` example. Save this source as `soda-runtime.can` in a separate development checkout:

```canlang
## Real team data, canonical file authorization and retained scheduled occurrence.
app SodaRuntime
Given
 Entry { label:text, attachment:file?, ticks:int=0 }
 policy Entry read=owner fields=label,attachment,ticks
 policy Entry read=members fields=label,ticks
 event Due { entry:Entry }
When
 crud Entry by=owner fields=label
 scenario attach(entry:Entry,attachment:file) -> bool by=owner
  do
   set entry {attachment=attachment}
   return true
 scenario arm(entry:Entry,due:datetime) by=owner
  do schedule "increment" at=due event=Due {entry=entry}
 scenario fire on=Due
  do set event.entry {ticks=event.entry.ticks+1}
Then
 page / title="Soda runtime integration"
  list Entry empty="No retained entries"
   text row.label,row.ticks
```

The owner can create entries, attach a file and schedule an increment. Members can read labels and counts, but the attachment field belongs to the owner. The page displays entries; this example's mutations use authenticated application operations, not an upload button. See Canlang's [permissions guide](https://github.com/veighnsche/canlang/blob/main/docs/public/02-Language/03-permissions.md) for the language rules.

### Prepare a separate bundle and private directories

Use a development bundle assembled from this source through [bundle assembly](#assemble-an-application-bundle), with separate development Identity and State resource IDs. In those compilation commands, use `soda-runtime.can` in place of `app.can`. Select a private development HTTPS origin, such as `https://files-dev.soda.test`, and arrange its DNS, routing and certificate trust with your administrator. Replace that example hostname with your actual development address.

Run the following in your Project account. Keep this directory outside the Git checkout. Choose a new name if `canlang-file-dev` already exists; continue only after creating a fresh directory:

```sh
umask 077
CANLANG_DEV_ROOT="$HOME/canlang-file-dev"
mkdir -m 0700 "$CANLANG_DEV_ROOT"
mkdir -m 0700 "$CANLANG_DEV_ROOT/data" "$CANLANG_DEV_ROOT/credentials"
```

Place the matching development bundle at `$CANLANG_DEV_ROOT/bundle.json`. Its `CAN_AUTH_ORIGIN` must exactly match the chosen HTTPS origin, without a path or trailing slash. This example's typed `Due` event is scheduled by `arm`; it does not declare an interval schedule, so its deployment descriptor has `schedules: []`. Keep any actual interval requirements when adapting the source.

### Create an app-local owner and member

Keep the runtime stopped during account administration. In a private editor, create `$CANLANG_DEV_ROOT/credentials/owner.json` with these fields, replacing the password placeholder with a unique development password:

```json
{
  "email": "owner@example.test",
  "password": "REPLACE_WITH_A_PRIVATE_PASSWORD",
  "timezone": "UTC"
}
```

The file must be owned by your account, regular rather than a symlink, at most 64 KiB and inaccessible to group and other users. Mode 0600 satisfies the permission requirement. Passwords contain 8–512 characters; `timezone` is optional, defaults to `UTC` and must be a supported timezone. Keep credentials out of Git, shell arguments, screenshots and shared logs.

```sh
chmod 0600 "$CANLANG_DEV_ROOT/credentials/owner.json"
can-soda bootstrap --bundle "$CANLANG_DEV_ROOT/bundle.json" \
  --data "$CANLANG_DEV_ROOT/data" \
  --credentials "$CANLANG_DEV_ROOT/credentials/owner.json"
```

Success returns `ok: true`, a `userId` and a `teamId`. Use the returned **teamId** for `TEAM_ID` below. Wait for a successful command exit before running another administration command or starting the runtime; a printed result alone does not confirm that its writer has finished.

An identical retry returns the saved claim; different credentials, an incomplete claim or populated but unclaimed Identity data cause refusal. Bootstrap does not reset an existing account or send verification email.

Create a separate private `$CANLANG_DEV_ROOT/credentials/member.json` file with exactly `email` and `password`, using `member@example.test` and a different unique password. Apply the same ownership and file permissions. Then add that account to the owner's team:

```sh
chmod 0600 "$CANLANG_DEV_ROOT/credentials/member.json"
can-soda member --bundle "$CANLANG_DEV_ROOT/bundle.json" \
  --data "$CANLANG_DEV_ROOT/data" --owner TEAM_ID \
  --credentials "$CANLANG_DEV_ROOT/credentials/member.json"
```

The result confirms an ordinary verified member; wait for successful command exit before serving the data. Matching verified accounts and active memberships are reused; this command neither resets credentials nor grants ownership. Removed memberships, unverified accounts and credential mismatches cause refusal.

### Serve through the development HTTPS origin

Have the administrator configure an HTTPS gateway to this Project's fixed loopback backend. Port 8080 is an example: choose an available port and use the same one in the gateway configuration and command. A gateway elsewhere needs an explicitly configured route to the Project; it cannot reach the Project's loopback address directly.

Forward the complete application origin, including `/auth`, `/mcp`, `/api/operations` and `/files`. Preserve bodies, methods, application cookies, Origin, CSRF headers and upload offsets. File grants use the external HTTPS origin; the backend below speaks HTTP. See [private networking and TLS](../40-Configure/50-private-networking-and-tls.md) for client access requirements.

From the same Project terminal, replace `TEAM_ID` with the bootstrap result and start the foreground supervisor:

```sh
can-soda serve --bundle "$CANLANG_DEV_ROOT/bundle.json" \
  --data "$CANLANG_DEV_ROOT/data" --owner TEAM_ID \
  --host 127.0.0.1 --port 8080
```

Successful startup reports `ok: true` and `command: "serve"`. Open the external HTTPS origin with an app-local account. Startup alone does not establish client DNS, certificate trust or file authorization. Files requires a fixed backend port; `--port 0` is refused. Keep only one runtime or administration writer on the data directory.

### Check access and retained results

```doc-pending
{
  "id": "native-files-client",
  "title": "Authenticated file-upload example",
  "message": "A complete client example for upload, finalization and attachment will be added once the supported authenticated client procedure is confirmed.",
  "guidance": "The example's page lists entries and has no upload button. The sequence below describes the required operations and expected results; it is not yet a runnable client walkthrough."
}
```

Use an authenticated application client for the example's operations:

1. Create an entry as the owner and retain its current ID and version.
2. Upload a small non-sensitive file through the `SodaRuntime.attach` upload intent and finish that upload.
3. Attach the finalized file to the entry using its current record reference. Upload completion and attaching a file are separate operations.
4. Check the owner/member results below with separate sessions. Before each scheduling check, record the current `ticks` value so one increment is recognizable.

| Check | Expected observation |
| --- | --- |
| Owner reads the entry's attachment | `/files/read/SodaRuntime.Entry/ENTRY_ID/attachment`, using the actual entry ID and owner session, returns HTTP 200 and the uploaded bytes. |
| Same-team member reads that attachment | The same path returns HTTP 404; permission to read the label and count does not expose the attachment. |
| Owner calls `SodaRuntime.arm` with the current entry and a future `due` time | The scheduled event increments `ticks` once; the page displays the updated count. |
| Stop before the scheduled time and restart after it | The entry and attachment remain; the retained occurrence increments the count once after restart. |

Keep owner and member sessions separate so a retained owner cookie cannot invalidate the permission check. A Forgejo sign-in does not supply either application account.

### Stop without removing development data

Press Ctrl-C in the serving terminal, or send SIGTERM to its supervisor, and wait for shutdown to finish. `can dev stop` controls the ordinary preview, not this `can-soda` process. Stop any separately started preview through its own command.

Restart with the same bundle, data directory, team and origin to inspect retained results. A storage-identity mismatch requires resolving the selected bundle and data; do not erase the data to suppress it. For a fresh experiment, choose a new private development root. Keep the existing data and credential files private, and remove only an experiment you have deliberately finished. Production storage and app-owner credentials never enter this development root.

## Maintain installed Project tools

Project tools persist in the retained Project root. Opening a terminal or importing an application image does not refresh them. Stop active Canlang processes before asking the host administrator for a supported [same-root Project maintenance](../50-Operate/60-project-maintenance.md) operation.

## Assemble an application bundle

For packaging, compile the reviewed `app.can` with the installed native catalog and retain its scenario receipts. From the application directory, choose a new `publish-app.json` output file:

```sh
can compile --format=json --native-scenario-receipts \
  --catalog=/opt/canlang/lib/node_modules/@canlang/cloudflare/dist/native-catalog.json \
  app.can > publish-app.json
```

Successful compilation produces the application artifact for the bundle. If compilation fails, correct the reported source or catalog problem before using the output.

```doc-pending
{
  "id": "canlang-publisher-inputs",
  "title": "Prepare publisher configuration",
  "message": "Steps for generating the deployment descriptor and preparing its matching environment file will be added once the supported publisher recipe is confirmed.",
  "guidance": "Compilation and ordinary preview can be used independently. The bundle command below requires both configuration files; compilation alone does not produce them."
}
```

Bundle assembly also needs Canlang's deployment descriptor and environment configuration. These inputs must describe the same application and target:

| Input | Required contents |
| --- | --- |
| `publish-app.json` | The compiled output of the reviewed application source, including native scenario receipts. |
| `deployment.json` | The matching application/source identity, artifact digest, producer compatibility information, required capabilities, resource bindings, secret references and declared schedules. |
| `soda-environment.json` | The target environment's resource assignments, names of available secrets and non-secret configuration, including `CAN_AUTH_ORIGIN`. |

The descriptor must preserve the application's declared requirements. Its artifact digest is SHA-256 of `JSON.stringify(parsedArtifact)`, rather than a checksum of the formatted `publish-app.json` file. `can-soda build` consumes the descriptor and environment files; compilation alone does not produce them.

Declare separate `DB` and `STATE_DB` bindings and map them to distinct resource IDs. `DB` holds app-local Identity; `STATE_DB` holds the team's application state. Production identities must be separate from preview and example data. Agree the private HTTPS origin with the operator before building; `CAN_AUTH_ORIGIN` is that origin without a path or trailing slash, on a hostname separate from Forgejo.

From the directory containing the three prepared files, choose a new output filename. The example uses `bundle.json`, which must not already exist:

```sh
can-soda inventory
can-soda build --artifact publish-app.json --descriptor deployment.json \
  --environment soda-environment.json --out bundle.json
```

Inventory reports supported providers and explicit rejections. Build checks the inputs and assembles the installed runtime modules. Success creates the portable JSON bundle and prints a result with `ok: true`, `command: "build"` and its `mixedSha256`. If a requirement or compatibility check fails, correct the named input or choose a supported target; do not remove genuine application requirements to bypass refusal.

The bundle retains the application, descriptor, environment, compatibility date and flags, storage fingerprint and module bytes. Keep its identity with those exact bytes. Environment variables in this file are non-secret configuration; never include owner passwords, provider credentials or development state. Bundle creation does not provision an owner, install an app or start a service.

## Export a prepared application image

```doc-pending
{
  "id": "canlang-image-build",
  "title": "Acquire runtime inputs and build the image",
  "message": "The matched runtime image, tools archive and application build recipe will be documented once their supported acquisition and build steps are confirmed.",
  "guidance": "The export command below requires an application image that has already been built and checked. Building the JSON bundle alone does not create that image."
}
```

The application publisher packages the exact bundle over the runtime image selected for the destination Soda release. The application layer retains the bundle's identities and the matching runtime/tools references. The operator uses these values during [application preparation](../50-Operate/40-private-applications.md#check-the-publisher-handoff); a tag alone does not establish compatibility.

On the publisher's build machine, after the application image has been built and checked, save that image from the Podman store that contains it. Replace `APP_IMAGE` with its exact local image ID and `APP_ARCHIVE` with a new absolute archive path under `/home`:

```sh
podman save --format oci-archive --output APP_ARCHIVE APP_IMAGE
```

The archive contains the image and its layers for the operator's `podman load` step. Use [Podman's image save operation](https://docs.podman.io/en/latest/markdown/podman-save.1.html); exporting a container filesystem does not preserve the same image metadata. Transfer the archive and its verified identity information through your team's trusted path. Application data, owner credentials and backups stay outside the image handoff.

## Continue to application operations

Once the publisher has supplied the prepared archive and its matching identity references, continue with [Private applications](../50-Operate/40-private-applications.md#check-the-publisher-handoff) to check that handoff, create production app-local accounts and explicitly install and start the independent application.

Private DNS and TLS are covered by [Private networking and TLS](../40-Configure/50-private-networking-and-tls.md); stopped backups and fresh restores belong to [Backups and restoration](../50-Operate/50-backups-and-restoration.md).
