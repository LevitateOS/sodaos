# Let an active employee view their assigned task

Extend `canViewTask(user, task)` so an active HR user can view any task and an
active employee can view a task assigned to their exact nonempty user ID.
All inactive users, other employees and employees viewing unassigned tasks
must be denied. Assume `user` and `task` are objects, `active` is boolean and
IDs/roles are strings or absent. Do not implement a new authentication system.

Add tests for an active matching assignee, an active different assignee, an
inactive matching assignee, an unassigned task, empty/missing IDs, and both
existing HR cases. A missing user ID and missing assignee ID must not compare
as an authorization match. Preserve the baseline tests.

Acceptance: `node --test access.test.mjs` passes, the diff stays within this
function/tests, independent review covers the current commit, and the PR meets
the configured required check and human approval policy. No deployment is
authorized by this issue.
