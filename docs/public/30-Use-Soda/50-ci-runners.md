# CI runners

Register local Forgejo or GitHub execution capacity through operator Cockpit while the provider owns workflows, scheduling and results.

Each local runner has one job slot, a noninteractive unprivileged Linux runtime
account and persistent working state. It is not a developer workspace. Jobs run
repository code with network access; use local capacity only for trusted
repositories and contributors. An unprivileged account is not a hostile-code
sandbox guarantee.

## Register a Forgejo runner

You need both host operator access to Cockpit and native Forgejo permission to
register the runner. These are independent authorities.

1. In Forgejo runner administration, create the runner registration and obtain
   its UUID and confidential token. Follow
   [Forgejo Actions administration](https://forgejo.org/docs/latest/admin/actions/).
2. Open **Cockpit → Runners → Create local runner** and choose **Bundled Forgejo**.
3. Enter a lowercase runner ID, the registration UUID/token and the intended
   `name:host` labels, such as `soda-linux:host`.
4. Select **Register and start**, then inspect local listener state and capacity.
5. Use the corresponding label, such as `runs-on: soda-linux`, in a trusted
   repository workflow and inspect the actual run in Forgejo.

The bundled Forgejo endpoint comes from the appliance configuration, not a
hostname/port inferred from the Cockpit browser URL. A listening local service
is not proof that the provider scheduled or passed a job.

## GitHub capacity

For [GitHub-hosted runners](https://docs.github.com/en/actions/using-github-hosted-runners),
configure the workflow in GitHub; no Soda local registration is needed.

For a local GitHub runner:

1. Obtain the native registration URL and short-lived token from the repository,
   organization or enterprise Actions settings. Follow
   [GitHub's self-hosted registration instructions](https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners/adding-self-hosted-runners).
2. Choose **GitHub** in **Create local runner** and enter the values, a runner ID
   and the desired custom labels.
3. Select **Register and start** and check its listener, architecture and capacity.
4. Match the provider's native labels in your workflow and verify a real job in GitHub.

Enter secrets only into the protected registration controls. Do not put them in
workflow files, screenshots, issue reports or shared logs. Native clients retain
the registration state they need in their dedicated runtime directories.

## Inspect and control a runner

The page shows provider, client version, listener state and configured slots.
Use **Start**, **Stop** or **Restart** for the selected local service. Let active
jobs finish first; stopping a listener can interrupt work.

For service failures inspect `soda-runner@RUNNER_ID.service` in Cockpit Services
and Logs. For expired tokens, missing labels or unscheduled jobs, inspect the
provider's registration, workflow and run history. Soda is not another scheduler.

## Remove or replace a runner

**Remove deletes local runner state**, including its runtime account, client
working files, dependencies and uncommitted job data. Preserve anything needed
before confirmation. Provider registration and history remain provider-owned;
remove an obsolete remote record separately.

For client replacement, stop after the current job, preserve local data, remove
the selected runner and review its provider registration. Obtain fresh native
registration input for the replacement and verify its client version and a real
job. Do not assume a host update replaced an existing runner's installed copy.

If an operation fails, retain its partial result and inspect local/provider
state before retrying. Registration may have changed one side without completing
the other. Do not repeatedly register or delete unrelated accounts as a repair.
