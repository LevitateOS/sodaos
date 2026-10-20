# Refresh tools in a retained Project

Refresh supported tools in an existing Project while preserving its accounts, files, checkouts and volumes. Choose an operation matching its symptom and running/idle requirements before invoking maintenance.

Same-root maintenance updates selected Project tools inside the existing Project container. Importing a new Project image or restarting the appliance does not refresh an existing writable root.

## Check operation requirements

Terminal-support repair can inspect an eligible running or stopped retained root, but it requires an idle Project with no active or uncertain managed terminal or Factory work. Muse tool maintenance requires an already running Project.

Canlang tool maintenance also requires a running Project, and you must stop Canlang previews, compilers and platform commands first; the operation refuses while a process is using `/opt/canlang`. Maintenance does not start a stopped Project or kill a process to make room.

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

   It updates the public Muse tools and launch interface in the same root.

4. To refresh Canlang tools, first stop previews, compilers and platform commands, then run this command against an already running Project:

   ```sh
   sudo /usr/libexec/soda/soda-project-maintain canlang --project PROJECT_ID
   ```

   It updates the Canlang tool installation in the same root without resetting preview state. For either Muse or Canlang, start a stopped Project through Project controls before maintenance.

## Recover an interrupted maintenance operation

If terminal-support reports that the Project is busy, keep its data, let managed work finish and rerun the same check for the same `PROJECT_ID`; apply the repair only when that inspection admits it. If terminal-support is interrupted, preserve its record and rerun the same exact command for that Project so it inspects the recorded operation and retained root before continuing. Do not remove the record or substitute another Project.

Canlang maintenance uses a directory rename and can leave the old or new tool directory in place after interruption. Before any retry, inspect `/opt/canlang`, any retained `/opt/.canlang-before-*` directory and the command's diagnostic output; do not assume automatic replay or copy one directory over another. If state is unclear, stop and follow [Recovery](80-recovery.md) with the exact Project identity. Preserve accounts, checkouts and preview data throughout.

The operation changes only the selected helper or tool installation in that existing root. It does not refresh the Project's OS image, replace the container, start stopped services, or upgrade the host. For a supported appliance release transition, see [Updates and fallback](50-updates-and-fallback.md).
