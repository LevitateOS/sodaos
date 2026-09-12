# CI runners

Register local Forgejo execution capacity through **Runners** in Forgejo's native
navigation (`/?soda-view=runners`) while Forgejo owns workflows, scheduling and results.
Sign in to Forgejo first. Only the configured Soda operator can manage local capacity.

Each local runner has one job slot, a noninteractive unprivileged Linux runtime
account and persistent working state. It is not a developer workspace. Jobs run
repository code with network access; use local capacity only for trusted
repositories and contributors. An unprivileged account is not a hostile-code
sandbox guarantee.

## Register a Forgejo runner

You need Soda operator authority for local registration. An authorized Forgejo
administrator separately creates the system runner record and supplies its UUID/token.
Being a Forgejo administrator does not grant Soda operator authority, and the
Soda operator does not need site-admin rights to control existing local listeners.

1. In Forgejo runner administration, create a system runner registration and obtain
   its UUID and confidential token. Follow
   [Forgejo Actions administration](https://forgejo.org/docs/latest/admin/actions/).
2. Open **Runners → Register local runner** in the native Forgejo page.
3. Enter a lowercase runner ID, the registration UUID/token and the intended
   `name:host` labels, such as `soda-linux:host`.
4. Select **Register and start listener**, then inspect local listener state and capacity.
5. Use the corresponding label, such as `runs-on: soda-linux`, in a trusted
   repository workflow and inspect the actual run in Forgejo.

The bundled Forgejo endpoint comes from the appliance configuration, not a
hostname/port supplied by the browser. A listening local service
is not proof that the provider scheduled or passed a job.

Enter secrets only into the protected registration controls. Do not put them in
workflow files, screenshots, issue reports or shared logs. Native clients retain
the registration state they need in their dedicated runtime directories.

## Inspect and control a runner

The page shows provider, client version, listener state and configured slots.
Use **start**, **stop** or **restart** for the selected local service and confirm
its exact ID. Start and Restart enable host-boot start; Stop disables it. Let active
jobs finish first; Stop and Restart can interrupt work. Configured slots are not
provider-available capacity. Refresh reports local observations, not job outcomes.

For service failures inspect `soda-runner@RUNNER_ID.service` in Cockpit Services
and Logs. For expired tokens, missing labels or unscheduled jobs, inspect the
provider's registration, workflow and run history. Soda is not another scheduler.

## Remove or replace a runner

**Remove deletes local runner state**, including its runtime account, client
working files, dependencies and uncommitted job data. Preserve anything needed
before confirmation. Provider registration and history remain provider-owned;
remove an obsolete remote record separately.

The listener uses the host's installed `forgejo-runner` package. Updating that
package is host maintenance; it does not require deleting a runner or registering
it again. Preserve registration, working state and active jobs during maintenance.

If an operation fails, retain its partial result and inspect local/provider
state before retrying. Registration may have changed one side without completing
the other. Do not repeatedly register or delete unrelated accounts as a repair.
