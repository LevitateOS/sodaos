import assert from 'node:assert/strict';
import {check, sha} from './support';
import {nativeSpacesMount} from '../../sodaspaces-journey-evidence';
import {observeMatrixShell} from '../../sodaspaces-matrix-native';
import {projectDiagnostic} from './project-phase';
let stage = 'not_started';
export const witnessDiagnostic = (error: any) => ({...projectDiagnostic(error), stage});

export function witnessCommand(operation: string, marker: string, relative: string) {
  check(/^[a-f0-9]{32}$/.test(operation) && /^SODA_PERSIST_[a-f0-9]{32}$/.test(marker), 'fixed_witness_identity');
  check(
    /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(relative) && !['.', '..'].includes(relative),
    'witness_relative_filename'
  );
  const content = 'Soda qualification retained member data ' + operation + '\n';
  // The exact joined member writes its own HOME over the real browser PTY.
  // Noclobber refuses a preexisting file, including a symlink. Nothing writes
  // this marker through SSH, a host mount, database or root process.
  const command =
    '(set -eu; login=$(id -un); case "$login" in root|""|*[!a-z0-9_-]*) exit 1;; esac; ' +
    'home=$(getent passwd "$login" | cut -d: -f6); test "$home" = "$HOME"; test "$home" = "/home/$login"; ' +
    'file="$home/' +
    relative +
    '"; test ! -e "$file"; test ! -L "$file"; ' +
    '(umask 077; set -C; printf "%s\\n" "Soda qualification retained member data ' +
    operation +
    '" > "$file"); ' +
    'test -f "$file"; test ! -L "$file"; test "$(stat -c %a "$file")" = 600; ' +
    'sum=$(sha256sum "$file"); sum=${sum%% *}; ' +
    'printf "\\n' +
    marker +
    ':{\\"login\\":\\"%s\\",\\"home\\":\\"%s\\",\\"sha256\\":\\"%s\\"}\\n" "$login" "$home" "$sum")';
  return {command, content, sha256: sha(content)};
}

// Call only after the real new browser TTY phase on the normally prepared Project. Its existing raw-run
// block and exact terminal WebSocket permit stay installed before navigation.
// The result contains the precise session alias consumed by Docker persistence.
export async function writePersistenceWitness(
  page: any,
  cfg: any,
  projectResult: any,
  save: (name: string, value: any) => Promise<any>
) {
  check(cfg.persistence_phase?.execution_released === true, 'persistence_witness_held');
  const operation = cfg.persistence_phase.operation_id;
  const session = projectResult.session,
    facts = projectResult.facts,
    project = projectResult.environment?.id;
  check(
    projectResult.observed === true &&
      /^p[a-f0-9]{24}$/.test(project) &&
      session?.environment === project &&
      /^[a-f0-9]{32}$/.test(session.id) &&
      facts?.tty === true &&
      facts.login === projectResult.join?.login,
    'original_browser_tty_required'
  );
  const marker = 'SODA_PERSIST_' + operation,
    {command, sha256} = witnessCommand(operation, marker, cfg.persistence_phase.witness_relative);
  const wire = observeMatrixShell(page);
  try {
    check(
      Number.isSafeInteger(projectResult.repository?.id) && projectResult.repository.id > 0,
      'browser_witness_repository'
    );
    stage = 'witness_navigate_workspace';
    await page.goto(cfg.guest.origin + '/workspace?repository=' + projectResult.repository.id);
    stage = 'witness_mount_workspace';
    await nativeSpacesMount(page);
    const owner = page.locator('soda-spaces'),
      projectButton = owner.locator('.soda-project-group[data-environment-id="' + project + '"] .soda-project-select');
    const directory = owner.getByRole('button', {name: 'Projects', exact: true});
    stage = 'witness_await_project_directory';
    await projectButton.or(directory).filter({visible: true}).first().waitFor();
    stage = 'witness_disclose_projects';
    if (!(await projectButton.isVisible())) await directory.click();
    stage = 'witness_select_project';
    await projectButton.click();
    stage = 'witness_select_terminals';
    await page.locator('soda-spaces').getByRole('button', {name: 'Terminals', exact: true}).click();
    stage = 'witness_disclose_workspace_options';
    await owner.getByLabel('Workspace options', {exact: true}).click();
    stage = 'witness_select_exact_session';
    await owner.locator('button[data-terminal-id="' + session.id + '"]:visible').click();
    stage = 'witness_wait_connected';
    await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
    stage = 'witness_read_original_shell';
    assert.deepEqual(await wire.shell(session, false), facts);
    // This retained command is the durable intent, written before its PTY input.
    stage = 'witness_save_intent';
    await save('witness.request.json', {
      project,
      session: session.id,
      operation_id: operation,
      relative: cfg.persistence_phase.witness_relative,
      sha256,
      command,
    });
    wire.clear(session.id);
    const screen = page.locator('.soda-workspace-terminal:visible .xterm-helper-textarea');
    stage = 'witness_focus_existing_tty';
    await screen.focus();
    stage = 'witness_send_owned_input';
    await page.keyboard.insertText(command);
    stage = 'witness_enter_owned_input';
    await page.keyboard.press('Enter');
    const deadline = Date.now() + 15000;
    let observed: any;
    do {
      const text = wire
        .text(session.id)
        .replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '')
        .replaceAll('\r', '');
      const line = text.split('\n').find((line) => line.startsWith(marker + ':'));
      if (line) {
        observed = JSON.parse(line.slice(marker.length + 1));
        break;
      }
      stage = 'witness_observe_owned_output';
      await new Promise((resolve) => setTimeout(resolve, 100));
    } while (Date.now() < deadline);
    check(
      observed?.login === facts.login && observed.home === '/home/' + facts.login && observed.sha256 === sha256,
      'browser_member_file_write_unconfirmed'
    );
    stage = 'witness_verify_same_shell';
    assert.deepEqual(await wire.shell(session, false), facts);
    const witness = {
      project,
      login: observed.login,
      home: observed.home,
      relative: cfg.persistence_phase.witness_relative,
      sha256,
      session: session.id,
    };
    stage = 'witness_save_result';
    await save('witness.json', witness);
    stage = 'complete';
    return witness;
  } catch (error) {
    await save('witness-diagnostic.json', witnessDiagnostic(error));
    throw error;
  } finally {
    wire.dispose();
  }
}
