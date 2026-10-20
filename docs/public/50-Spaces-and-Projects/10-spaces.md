# Spaces and Workspace

Spaces brings repository Projects, Factory activity, applications, and your own terminal sessions together beside native Forgejo pages. Use it to find the right work and keep a live session available as you browse.

Open the global Workspace to see work across repositories, or open Workspace from a repository to focus its Feed. The persistent Workspace panel shares the same session owner as the full Workspace page. Repository selection is a viewing context: it does not create or start a Project, Join you to it, or authorize Factory work. For Project lifecycle and access, see [Projects and workspaces](20-projects-and-workspaces.md).

The workspace can retain several kinds of context at once. The repository you are browsing, the repository attached to a saved build request, and the Project and account bound to an existing terminal are separate targets. Changing one does not silently change the others. Before acting, check the repository named on the page or request and the account and Project shown for a terminal.

Use the Applications area to discover private applications available on the appliance. Opening the listing or an application link does not install or start an app. Application installation, lifecycle and app-local sign-in are handled separately in [Private applications](../70-Operate/40-private-applications.md).

Workspace views do not own the work they display. A split adds another view of the same session, and hiding or closing a panel leaves its shell or Factory run in place. Reconnecting targets the retained session; it does not launch a replacement shell or resume a finished AI conversation. Factory output is observable only with current code-write access. Watching a run does not give terminal input or takeover authority. Human terminal inventory is limited to your own sessions.

When you return to native Forgejo pages, the Workspace panel preserves its live terminal attachment. Browsing another repository never retargets a terminal's Project, Linux account or session. If access expires, sign in again as the same native account and inspect the original workspace entry before retrying an action whose result is unclear.

To follow and discuss Factory work, use the [Factory Feed](../40-Factory/70-factory-feed.md). To open your own browser terminal, follow [Workspace terminals](40-workspace-terminals.md). To connect an editor over SSH, see [Connect and develop](../60-Develop/20-connect-and-develop.md). These actions keep their own authority and lifecycle; navigating Workspace alone does not perform them.
