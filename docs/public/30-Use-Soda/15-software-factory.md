# Run your first software factory task

Use the factory operator interface to admit a Forgejo issue, run a bounded coding task, and inspect its verified pull request before merging it yourself.

## Prepare the factory

Use a private repository on this appliance's Forgejo. The authorizing person must
have write access. Configure separate non-administrator implementation and review
bot accounts, and protect the default branch through Forgejo's native settings.
The implementation worker cannot merge or publish to arbitrary branches; the
review worker can submit findings for its assigned pull request and commit.

The operator prepares the `soda-factory` command and a private configuration file.
It selects the Forgejo origin, repository ID, authorizing human's token file,
implementation and review actors and token files, and a Forgejo Actions workflow.
It also selects the agent version and model, pinned workspace and proxy images,
permitted network destinations, and CPU, memory, process and writable-storage
limits. Run the command as an unprivileged Linux operator with rootless Podman,
user namespaces, cgroup v2 and SELinux.

Keep configuration and credential files private with mode `0600`; factory state,
workspace and publication directories use mode `0700`. Supply secrets through
restricted files, never issues, command arguments or logs. Factory credentials
are separate from human project credentials. See
[People and access](../50-Operate/10-people-and-access.md#factory-identities-and-provider-access).

The initial account-auth profile uses Codex with a dedicated enrolled credential
home. Use the selected profile's supported private automation and enrollment
procedure. Account reuse lasts until renewal or revocation requires intervention;
it is not universal or permanent login. Soda does not silently switch to paid
API billing if the subscription becomes unavailable. Other harnesses and provider
accounts require their own qualified profile.

Configure the selected CI workflow to evaluate candidate pull requests on
`pull_request` events for `opened` and `synchronize`. Do not trigger additional
candidate evaluations from review or agent comments. CI remains Forgejo Actions;
Soda observes its result instead of adding another CI queue. See
[CI runners](50-ci-runners.md#verify-factory-candidates).

## Authorize a clear objective

Create an issue describing the expected behavior, relevant context and useful
acceptance checks. Resolve ambiguity before assigning work. Repository content
and issue comments are task input, not permission to change the execution policy.
Labels alone do not authorize a run.

The authorized operator admits the issue explicitly:

```sh
soda-factory --config /home/soda-tester/factory/config.json admit 12 issue-12-attempt-1
```

Replace the private configuration path, issue number and delivery ID with your
own values. Admission returns an attempt ID and records the objective, source
revision, authorizing person and policy. Repeating the same delivery ID returns
the existing attempt; it does not create another execution or reset its limits.
A different delivery is not admitted while the earlier attempt or cleanup remains
active.

## Run and inspect the attempt

Use the returned attempt ID:

```sh
soda-factory --config /home/soda-tester/factory/config.json run ATTEMPT_ID
soda-factory --config /home/soda-tester/factory/config.json status ATTEMPT_ID
```

Implementation starts from a fresh checkout. The agent edits, tests and commits
locally. A separate publisher checks the candidate and publishes only the assigned
branch, `soda/factory/ATTEMPT_ID`, then creates its pull request. Forgejo write
credentials stay outside the agent workspace. The worker cannot change protected
CI or policy paths.

CI and a fresh review workspace check the candidate's exact commit. The reviewer
receives the requirement and repository context, with a separate execution identity
and clean environment. It cannot publish implementation changes. A successful
agent process is not proof of correct software; inspect the test and review evidence.

If the first candidate has a repairable failure, one repair run may produce a
new commit. That commit needs new CI and a new fresh review; earlier evidence
cannot verify it. A further failure returns the attempt to a person.

## Understand time limits and outcomes

The initial limits are 90 minutes for implementation, 30 minutes for each review
or repair, one repair, four agent executions and two CI evaluations. The entire
attempt has a three-hour elapsed deadline, including waiting. One work item has
one active execution, and a Codex credential stream is used by one execution at
a time. CPU, memory, process, disk and network limits are enforced outside the agent.

Run outcomes are `succeeded`, `failed`, `cancelled` and `needs-human`. Cleanup
completion is recorded separately: a finished execution is not proof that all
resources have been removed. Forgejo comments and `soda:` labels summarize the
outcome and any intervention; the local status record includes run IDs and cleanup.

Expired limits, unavailable authentication or unresolved ambiguity require human
intervention. Repeated events and agent comments cannot silently restart the work.
See [Administration](../50-Operate/20-administration.md#operate-factory-runs)
for interrupted execution and reporting after an outage.

## Review and merge yourself

Inspect the pull request's final head commit, implementation summary, CI result
and fresh review. Confirm that all verification refers to that same commit and
that the native protected-branch requirements are satisfied. If the candidate
changes, obtain new verification before merging.

Merge through Forgejo using your human account. Soda does not automatically
merge, deploy the result to production or promote running service data. See
[Collaboration](35-collaboration.md#review-and-merge).

## Cancel or start a new explicit attempt

```sh
soda-factory --config /home/soda-tester/factory/config.json cancel ATTEMPT_ID
```

Cancellation withdraws publication authority, terminates the workspace's processes
and reconciles its recorded resources. Closing the issue also withdraws further
work when Soda checks its state. Confirm cleanup with `status`.

After resolving an intervention and completing cleanup, a human can admit the
issue with a new delivery ID, such as `issue-12-attempt-2`. This is a new explicit
attempt, not a continuation that erases the earlier outcome. Running an already
terminal attempt reports its outcome without launching another agent.

Persistent human projects remain available for investigation and manual development.
See [Projects and workspaces](20-projects-and-workspaces.md) and
[Connect and develop](../40-Develop/10-connect-and-develop.md). Do not reuse their
writable roots or credentials as disposable factory resources.
