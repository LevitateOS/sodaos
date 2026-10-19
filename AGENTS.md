# Working on SodaOS

SodaOS is unreleased and has zero users. Deliver working features quickly. Bureaucracy and red tape are our biggest enemies. Optimize for the current task and local use, not hypothetical future requirements. These rules apply to Codex and Muse; the owner's current instructions lead and override conflicting project or skill procedures.

## Implement and keep moving

- Own the full authorized outcome. Make the smallest useful change, get it working locally, fix actual failures and continue until the task is handled.
- Prefer fixing observed bugs over building elaborate defenses against bugs we might encounter later. Cross those bridges when the problem exists.
- Do not add reproducibility work, release machinery, compatibility layers, migrations, speculative recovery systems, abstractions or frameworks unless the current task actually requires them.
- Do not turn implementation into a planning audit, status reconciliation, architecture exercise or documentation cleanup. Stale plans and procedural requirements must not block useful work.
- Resolve routine implementation and setup choices yourself. Do not introduce approval gates, exception paperwork, bookkeeping commits or repeated requests for permission already given.
- Keep one current implementation. Replace obsolete code and its callers together; remove dead paths instead of preserving them for imaginary users.
- Reuse existing code, mature tools and working outputs. Test an uncertain assumption cheaply before investing in an approach that depends on it.
- Never inspect, compare or gate work on tool or dependency version numbers. Try the installed tool's required behavior and fix actual failures; do not downgrade working tools to satisfy stale guidance.

## Check that it works

- Run the smallest useful local check of the changed behavior. Once it works, move on. Add tests when they help catch a concrete failure, not to mirror the implementation or satisfy a ritual.
- Do not verify the verification, review reviews, rerun passing checks without a relevant change, or demand proof beyond the requested working behavior.
- Fix the actual failure and retest the affected behavior. Preserve unrelated passing results; do not restart the whole process.
- Local success is enough for local implementation. When the owner asks for a VM, installed app or deployment, check that actual path rather than substituting source tests or mock output.
- Describe what actually ran and what remains broken. Do not invent completion percentages or present a request, launch or report as a working result.

## Orchestrate without ceremony

- For an authorized Muse run: save the needed implementation instructions, start Muse, deliver the prompt, submit it, confirm it started and promptly give the owner the tmux command and requested waker prompt.
- Keep required model and writer checks brief. Do not delay the handoff with repeated settings inspections, log archaeology or runtime-internals probing after sufficient startup information exists.
- Necessary plans, prompts, checklists and run references support continuity; writing them down is not bureaucracy. Keep them useful and concise, and update existing records rather than multiplying trackers.
- Review asynchronously. After handing off completed work, agents continue the next ready task; pending review is not an approval gate.
- On a supervision wake, inspect actual progress, review available work and act on concrete blockers. Correct real tool errors, identity mismatches and stalls; do not repeatedly report them without taking the next useful action.
- Reuse active sessions and command handles when recovering the current run. Coordinate actual writer overlaps briefly, preserve unfinished work and continue independent work.
- When the owner owns scheduling, create no schedules or timers. The requested waker sends one poke to Codex and ends naturally; Codex performs supervision.

## Respect the workspace and actual boundaries

- Work in `~/Projects/sodaos`; create worktrees only when requested. Preserve unrelated edits and leave the predecessor/Updates platform alone.
- All writes, builds, caches, state and scratch belong under `/home`. Do not use the root disk or `/tmp` as a fallback.
- Protect credentials, retained data and existing human work. Never expose secrets in commands, logs or screenshots. Do not reinstall a VM, overwrite newer data, rewrite history or clean unrelated resources without authorization.
- Implementation includes the code, relevant tests, necessary documentation and configuration. Deployment requests include the needed builds, transfers, scoped restarts and actual local checks. Do not infer permission to publish, push, open PRs, merge remotely or contact others.
- Follow [language ownership](docs/development/ownership.md): Go for network servers, Rust for system/privileged applications, no authored Python. Put changes in the existing responsible package and preserve dependency direction and canonical definitions.
- Read only the guidance relevant to the task: [docs index](docs/README.md), [testing](docs/development/testing.md), [native support](docs/development/native-support.md) and [installation](docs/guides/installation.md). Correct documentation your change makes misleading; link to existing owners instead of duplicating them.
- Keep coordination logs, consultation transcripts and temporary process artifacts outside the repository, including its ignored directories. Keep necessary run references under `/home`; technical decisions belong in their existing owning docs.
- JEV/SystemOne usage is free. Agents should consult it often for substantive implementation choices, uncertainty and blockers; do not ration calls for cost or wait for a separate request. Supply the relevant context because it cannot research. The owner confirms this project is open source and authorizes sending project code/context to TypeSafe's JEV API for processing and design decisions; exclude credentials and user data.
- For each consultation, make three independent calls asking the same underlying question in different prose. Keep the facts, alternatives and decision criteria identical. Write all three prompts before reading any answers, deliberately counterbalance your preferred answer's framing, and do not feed earlier responses into later calls. Compare the results and investigate disagreement rather than cherry-picking support. This reduces framing bias; agreement is advice, not proof that an answer is correct.
- If genuinely blocked, identify the exact missing input or permission, explain what is needed and continue everything independent. Communicate useful progress and results concisely.
