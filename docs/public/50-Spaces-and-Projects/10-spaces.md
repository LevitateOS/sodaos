# Find work in Spaces and Workspace

Spaces brings repository Projects, Factory work, private applications, and human terminals beside Forgejo. Use Workspace to find work and keep your session available as you browse.

## Choose a workspace view

Open the global Workspace to see activity across repositories, or open Workspace from a repository to focus its Feed. The persistent Workspace panel shares the same session owner as the full Workspace page.

Repository selection changes the viewing context only. It does not create or start a Project, Join you to it, or authorize Factory work. Use [Projects and workspaces](20-projects-and-workspaces.md) for those actions.

## Keep repository and session targets clear

The repository you are browsing, the repository attached to a saved build request, and the Project and account bound to an existing terminal are separate targets. Changing one does not silently change the others. Before acting, check the repository named on the page or request and the account and Project shown for a terminal.

Native Forgejo navigation preserves the live terminal attachment in the Workspace panel. Browsing another repository never retargets a terminal's Project, Linux account or session. If access expires, sign in again as the same native account and inspect the original workspace entry before retrying an action whose result is unclear.

## Find applications and Factory activity

Use the Applications area to discover private applications available on the appliance. Opening the listing or an application link does not install or start an app. Application installation, lifecycle and app-local sign-in are covered in [Private applications](../70-Operate/40-private-applications.md).

To follow and discuss Factory work, use the [Factory Feed](../40-Factory/70-factory-feed.md). Factory output is observable only with current code-write access. Watching a run does not provide terminal input or takeover authority.

## Manage session views

Workspace views do not own the work they display. A split adds another view of the same session, while hiding or closing a panel leaves its shell or Factory run in place. Reconnecting targets the retained session; it does not launch a replacement shell or resume a finished AI conversation. Human terminal inventory is limited to your own sessions.

To open your own browser terminal, follow [Workspace terminals](40-workspace-terminals.md). To connect an editor over SSH, see [Connect and develop](../60-Develop/20-connect-and-develop.md). Those actions keep their own authority and lifecycle; navigating Workspace alone does not perform them.
