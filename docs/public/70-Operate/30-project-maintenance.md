# Refresh tools in a retained Project

Same-root maintenance updates selected Project tools inside the existing Project container. It preserves the Project's accounts, files, checkouts and volumes; importing a new Project image or restarting the appliance does not refresh an existing writable root.

Maintenance prerequisites depend on the operation. Terminal-support repair can inspect an eligible running or stopped retained root, but it requires an idle Project with no active or uncertain managed terminal or Factory work. Muse tool maintenance requires an already running Project. Canlang tool maintenance also requires a running Project, and you must stop Canlang previews, compilers and platform commands first; the operation refuses while a process is using `/opt/canlang`. Maintenance does not start a stopped Project or kill a process to make room.

Run the exact supported operation as appliance root, using the Project ID shown in Soda. For terminal support, inspect first, then apply only if the check identifies the need:

```sh
sudo /usr/libexec/soda/soda-project-maintain terminal-support --project PROJECT_ID --check
sudo /usr/libexec/soda/soda-project-maintain terminal-support --project PROJECT_ID
```

Replace `PROJECT_ID` with the exact Project identifier. For Muse or Canlang tools, run the corresponding command against that same retained Project:

```sh
sudo /usr/libexec/soda/soda-project-maintain muse --project PROJECT_ID
sudo /usr/libexec/soda/soda-project-maintain canlang --project PROJECT_ID
```

## Recover an interrupted maintenance operation

If terminal-support reports that the Project is busy, keep its data, let managed work finish and rerun the same check for the same `PROJECT_ID`; apply the repair only when that inspection admits it. If terminal-support is interrupted, preserve its record and rerun the same exact command for that Project so it inspects the recorded operation and retained root before continuing. Do not remove the record or substitute another Project.

Canlang maintenance uses a directory rename and can leave the old or new tool directory in place after interruption. Before any retry, inspect `/opt/canlang`, any retained `/opt/.canlang-before-*` directory and the command's diagnostic output; do not assume automatic replay or copy one directory over another. If state is unclear, stop and follow [Recovery](80-recovery.md) with the exact Project identity. Preserve accounts, checkouts and preview data throughout.

The operation changes only the selected helper or tool installation in that existing root. It does not refresh the Project's OS image, replace the container, start stopped services, or upgrade the host. For a supported appliance release transition, see [Updates and fallback](50-updates-and-fallback.md).
