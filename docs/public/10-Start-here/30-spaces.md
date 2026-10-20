# Find work in Spaces and Workspace

Spaces brings repository Projects, Factory work, private applications, and human terminals beside Forgejo. Use Workspace to find work and keep your session available as you browse.

## Choose a workspace view

Open the global Workspace to see activity across repositories, or open Workspace from a repository to focus its Feed. The persistent Workspace panel shares the same session owner as the full Workspace page.

Repository selection changes the viewing context only. It does not create or start a Project, Join you to it, or authorize Factory work. Use [Projects and workspaces](../30-Projects-and-development/10-projects-and-workspaces.md) for those actions.

## Pick up daily work

| You want to… | Open |
| --- | --- |
| Find work or a question that needs you | [Follow Feed evidence](../20-Factory/20-factory-feed.md#follow-work-and-inspect-evidence), then [read and answer the thread](../20-Factory/20-factory-feed.md#comment-and-route-a-request) |
| Understand why an issue waits | [Identify the blocker and its owner](../20-Factory/80-factory-intervention.md#identify-the-actual-blocker) |
| Open or return to your own shell | [Open or reconnect to your terminal](../30-Projects-and-development/20-workspace-terminals.md#open-or-reconnect-to-your-terminal) |
| Start or revise a saved request | [Describe and save its direction](../20-Factory/30-plan-new-work.md#describe-the-objective) |
| Review a candidate or delivered result | [Inspect the candidate](../20-Factory/50-review-and-merge.md#inspect-the-candidate) and [confirm delivery](../20-Factory/50-review-and-merge.md#confirm-delivery) |
| Find a decision or past answer | [Search the library and assess its evidence](../20-Factory/70-knowledge-and-decisions.md#search-and-assess-what-you-find) |
| Recover an interrupted operation | [Find the recovery procedure](../50-Operate/80-recovery.md) |
| Make your first human contribution | [First connection](40-first-connection.md#make-a-first-contribution) |
| Prepare tools before automation | [Project preparation](../40-Configure/30-prepare-project.md) |

## Keep repository and session targets clear

The repository you are browsing, the repository attached to a saved build request, and the Project and account bound to an existing terminal are separate targets. Changing one does not silently change the others. Before acting, check the repository named on the page or request and the account and Project shown for a terminal.

Native Forgejo navigation preserves the live terminal attachment in the Workspace panel. Browsing another repository never retargets a terminal's Project, Linux account or session. If access expires, sign in again as the same native account and inspect the original workspace entry before retrying an action whose result is unclear.

## Find applications and Factory activity

Use the Applications area to discover private applications available on the appliance. Opening the listing or an application link does not install or start an app. Application installation, lifecycle and app-local sign-in are covered in [Private applications](../50-Operate/40-private-applications.md).

To follow and discuss Factory work, use the [Factory Feed](../20-Factory/20-factory-feed.md).
Feed entries and native records require current repository visibility; reading
them does not require Project Join. Each linked source or artifact retains its
own access checks.

[Live Factory terminal observation](../30-Projects-and-development/20-workspace-terminals.md#observe-factory-work)
requires current code-write access. Watching a run does not provide terminal
input or takeover authority. Opening your own shell requires explicit Join;
[People and access](../40-Configure/10-people-and-access.md#choose-authority-for-the-action)
explains the separate permissions.

## Manage session views

Workspace views do not own the work they display. A split adds another view of the same session, while hiding or closing a panel leaves its shell or Factory run in place. Reconnecting targets the retained session; it does not launch a replacement shell or resume a finished AI conversation. Human terminal inventory is limited to your own sessions.

To open your own browser terminal, follow [Workspace terminals](../30-Projects-and-development/20-workspace-terminals.md). To connect an editor over SSH, see [Connect and develop](../30-Projects-and-development/40-connect-and-develop.md). Those actions keep their own authority and lifecycle; navigating Workspace alone does not perform them.
