# First factory issue fixture

A small starting repository for the [review and approval walkthrough](https://sodaos.dev/use-cases/review-and-approval).
This is an exercise, not evidence of a completed autonomous factory run.

Copy `access.mjs`, `access.test.mjs` and `ISSUE.md` into a new disposable Git
repository. Use Node.js with its built-in test runner; no packages are needed.
From that new repository:

```sh
node --test access.test.mjs
git init
git add access.mjs access.test.mjs ISSUE.md
git commit -m "Add first factory issue baseline"
git rev-parse HEAD
```

Use your configured Git identity and an authorized private Forgejo remote.
Push the baseline there and configure its trusted CI workflow to run
`node --test access.test.mjs` on the actual PR candidate. Confirm the native
check's name and result before requiring it in the factory policy. The baseline
tests should pass; they intentionally do not yet cover the new assigned-user
behavior. `ISSUE.md` is the task to adopt, not a claim that it is already solved.

Enroll the repository, capacity and provider sponsorship through the selected
release's documented flow. Adopt the issue brief and observe its candidate,
separate review, corrections and current checks. In a human-gated policy,
confirm that agent review and green checks alone cannot authorize the merge.
Then supply the required human decision and confirm the native result. Do not
change policy just to make the exercise complete.

Retain release/source identity, baseline commit, issue/attempt/run references,
candidate commits, CI logs/results, review and approval records, and the actual
target commit after merge. Record failures and missing observations. Repeat a
failed-check and cancellation case before broadening access. Never fill this
record with invented output or turn this fixture into production access control
without the application's own identity and threat-model review.
