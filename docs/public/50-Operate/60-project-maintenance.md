# Refresh tools in a retained Project

Refresh supported tools in an existing Project while preserving its accounts, files, checkouts and volumes. Choose an operation matching its symptom and running/idle requirements before invoking maintenance.

Same-root maintenance updates selected Project tools inside the existing Project container. Importing a new Project image or restarting the appliance does not refresh an existing writable root.

## Check operation requirements

Terminal-support inspection accepts an eligible running or stopped retained root. Replacing support requires an idle Project with no active or uncertain managed terminal or Factory work. Muse tool maintenance requires an already running Project.

Choose the Project's exact ID, beginning with `p` followed by 24 lowercase hexadecimal characters. Keep its container identity and current running/stopped state with the maintenance result. A disconnected managed terminal or outstanding reservation still counts as work; closing its browser tab does not make repair safe.

Canlang tool maintenance also requires a running Project, and you must stop Canlang previews, compilers and platform commands first; the operation refuses while a process is using `/opt/canlang`. Maintenance does not start a stopped Project or kill a process to make room.

Canlang accepts only `--project PROJECT_ID`; it has no `--check` or tool-download option. It uses the appliance's admitted offline tools payload. Missing or mismatched payloads require correcting that input, not supplying a different archive on the command line.

## Run a supported refresh

Run the exact supported operation as appliance root, using the Project ID shown in Soda. Replace `PROJECT_ID` in these examples with the exact retained Project identifier.

1. Inspect terminal support without changing it:

   ```sh
   sudo /usr/libexec/soda/soda-project-maintain terminal-support --project PROJECT_ID --check
   ```

   The check observes the exact container and support bytes. It does not install them.

2. If the check admits repair and the Project remains idle, run the same operation against the same ID:

   ```sh
   sudo /usr/libexec/soda/soda-project-maintain terminal-support --project PROJECT_ID
   ```

   The operation updates the helper in that retained root and leaves a stopped Project stopped.

3. To refresh Muse tools, use this command only for an already running Project:

   ```sh
   sudo /usr/libexec/soda/soda-project-maintain muse --project PROJECT_ID
   ```

   It updates the public Muse tools and launch interface in the same root. Full maintenance also installs `dbus-broker` when missing and starts `dbus.socket`, which the managed launch path needs. It does not enroll a provider account or grant subscription use.

4. To refresh Canlang tools, first stop previews, compilers and platform commands, then run this command against an already running Project:

   ```sh
   sudo /usr/libexec/soda/soda-project-maintain canlang --project PROJECT_ID
   ```

   It updates the Canlang tool installation in the same root without resetting preview state. For either Muse or Canlang, start a stopped Project through Project controls before maintenance.

## Verify the selected refresh

Keep the command's result with the exact Project ID. For terminal-support repair, rerun its `--check` form and confirm the retained root now passes inspection; a Project that was stopped remains stopped. For Muse or Canlang, use the refreshed tool from a new invocation in that same running Project and verify the operation that needed repair. Existing processes do not become refreshed tools merely because files changed.

Interpret the terminal-support output by the reported phase. These are literal output phrases; the Project ID varies:

| Output phrase | Meaning and next action |
| --- | --- |
| `ready` with `read-only preflight; runtime compatibility checked only during maintenance` | The observed helper bytes match. The check did not install anything or execute the repair compatibility probe. |
| `support mismatch; explicit maintenance required` | The helper differs. Keep the Project idle and run the explicit repair; a successful check is not a completed repair. |
| `interrupted operation requires reconciliation` | A prior maintenance record remains. Preserve it and use the same repair command for that exact Project; inspection alone does not finish it. |
| `terminal support ready` | The repair path verified the helper and completed its recorded cleanup. Confirm the Project's intended running/stopped state before reopening work. |

Muse and Canlang maintenance return exit status `0` on completion; they do not require a success banner. A failure returns `1` with a `soda-project-maintain:` diagnostic. Keep the error and inspect the affected input or retained root before retrying. Canlang's old tool directory remains for explicit operator inspection even after success; its presence alone does not mean maintenance failed.

For example, when Canlang maintenance refuses because a preview is using `/opt/canlang`, stop that preview through its normal development controls and rerun maintenance for the same Project. Keep its source and preview data; the refusal is not a request to replace the Project. See [Canlang development](../30-Projects-and-development/70-canlang-development.md) for tool verification.

## Recover an interrupted maintenance operation

If terminal-support reports that the Project is busy, keep its data, let managed work finish and rerun the same check for the same `PROJECT_ID`; apply the repair only when that inspection admits it. If terminal-support is interrupted, preserve its record and rerun the same exact command for that Project so it inspects the recorded operation and retained root before continuing. Do not remove the record or substitute another Project.

For interrupted Muse maintenance, preserve the same root and inspect which public-tool or launch-interface stage failed. Its file replacements and interface attachment are separate effects; a failed command does not promise that all previous tools were restored. After resolving the reported cause and confirming the same Project is running, rerun the same full Muse command, then verify a fresh tool invocation. Startup's interface restoration is not a substitute for the full tool refresh.

Canlang maintenance uses a directory rename and can leave the old or new tool directory in place after interruption. Before any retry, inspect `/opt/canlang`, any retained `/opt/.canlang-before-*` directory and the command's diagnostic output inside that exact Project.

The previous directory is named for the selected archive SHA-256; an already occupied previous-directory destination blocks another attempt, including after a successful refresh to the same archive. Do not delete it merely to make a retry pass, assume automatic replay or copy one directory over another. If state is unclear, stop further maintenance and collect the exact Project identity, retained directory observations and command result in the [diagnostic worksheet](10-troubleshooting.md#collect-a-diagnostic-worksheet) for the host administrator. Preserve accounts, checkouts and preview data throughout.

The operation changes the selected helper or tool installation and its required launch support in that existing root. It does not refresh the Project's OS image, replace the container, start a stopped Project or upgrade the host. For a supported appliance release transition, see [Updates and fallback](70-updates-and-fallback.md).
