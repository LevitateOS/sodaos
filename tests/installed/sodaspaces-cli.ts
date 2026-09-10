// Actual selected CLI/browser-versus-SSH scenarios. Never installs tools, adds
// credentials, rewrites profiles or infers agent Working/Waiting/Finished states.
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import type {MatrixInput} from './sodaspaces-matrix-input';
import type {MatrixSession, MatrixFacts} from './sodaspaces-workspace-journey';
import {matrixSSH, restrictedText, sshArgs} from './sodaspaces-matrix-native';
import {object} from './sodaspaces-input';
function objectContext(text: string, term: string) {
  const context = object(JSON.parse(text)); assert(typeof context.tmux === 'string');
  assert.deepEqual(context.terminfo, [true, true]);
  return {tmux: context.tmux, browser_term: term, ssh_term: 'xterm-256color', terminfo: 'both installed'};
}
export function cliProtocolObservation(wire: string, expected: string) {
  return {expected_output: wire.includes(expected), unicode: /[^\x00-\x7f]/.test(wire),
    cursor: /\x1b\[[\d;]*[ABCDEFGHJKSTf]/.test(wire), alternate_screen: /\x1b\[\?(?:1049|1047)h/.test(wire),
    mouse_mode: /\x1b\[\?(?:1000|1002|1003|1006)h/.test(wire), bracketed_paste: /\x1b\[\?2004h/.test(wire)};
}
export async function exerciseSelectedCLIs(page: Page, request: MatrixInput, actor: number, sessions: readonly (MatrixSession & {facts?: MatrixFacts})[],
  stream: {text(id: string): string; metrics(id: string): {bytes: number; chunks: number}; clear(id: string): void}, checkActive: () => void) {
  assert.equal(request.provider_use, 'browser-and-ssh-for-declared-clis'); assert.equal(request.cli.length, 3);
  const results: unknown[] = [];
  for (const [index, scenario] of request.cli.entries()) {
    checkActive(); const session = sessions[index * 2], project = request.projects.find(project => project.environment === session?.environment); assert(session?.facts && project);
    const prompt = (await restrictedText(scenario.prompt_file, 4096)).trim();
    // Test prompts are non-secret fixture input, not credential channels. A marker
    // echoed from the prompt must not masquerade as generated output.
    assert(prompt && !/[\x00-\x08\x0b-\x1f\x7f]/.test(prompt) && !prompt.includes(scenario.expected_text));
    assert.equal(matrixSSH(request, project, actor, scenario.tool + ' --version'), scenario.version);
    const nativeContext = objectContext(matrixSSH(request, project, actor, `python3 -I -c 'import subprocess,json; print(json.dumps({"tmux":subprocess.check_output(["tmux","-V"],text=True).strip(),"terminfo":[subprocess.run(["infocmp",t],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0 for t in ["screen-256color","xterm-256color"]]}))'`), session.facts.term);
    await page.getByRole('button', {name: 'Sessions', exact: true}).click();
    await page.getByRole('button', {name: 'All', exact: true}).click();
    await page.locator('.soda-session-list button').filter({hasText: session.name}).click();
    const screen = page.locator('.soda-workspace-terminal:visible .xterm-helper-textarea'); await screen.focus();
    stream.clear(session.id); await page.keyboard.insertText(scenario.tool + ' --version'); await page.keyboard.press('Enter');
    let deadline = Date.now() + 15000;
    while (!stream.text(session.id).includes(scenario.version) && Date.now() < deadline) await new Promise(resolve => setTimeout(resolve, 100));
    assert(stream.text(session.id).includes(scenario.version), 'Browser CLI version not confirmed');
    stream.clear(session.id); await page.keyboard.insertText(scenario.tool); await page.keyboard.press('Enter');
    deadline = Date.now() + 15000;
    while (!stream.text(session.id).includes(scenario.ready_text) && Date.now() < deadline) {checkActive(); await new Promise(resolve => setTimeout(resolve, 100));}
    assert(stream.text(session.id).includes(scenario.ready_text), 'Declared CLI readiness not observed; no prompt submitted'); checkActive();
    await screen.evaluate((element, text) => {
      const data = new DataTransfer(); data.setData('text/plain', text);
      element.dispatchEvent(new ClipboardEvent('paste', {clipboardData: data, bubbles: true, cancelable: true}));
    }, prompt);
    await page.keyboard.press('Enter');
    deadline = Date.now() + 90000;
    while (!stream.text(session.id).includes(scenario.expected_text) && Date.now() < deadline) {checkActive(); await new Promise(resolve => setTimeout(resolve, 250));}
    const browser = cliProtocolObservation(stream.text(session.id), scenario.expected_text);
    assert(browser.expected_output, 'Browser CLI fixture output not confirmed');
    const browserStream = stream.metrics(session.id); assert(browserStream.bytes >= scenario.minimum_output_bytes && browserStream.chunks > 1, 'Declared browser streaming volume not observed');
    const box = await page.locator('.soda-workspace-terminal:visible .xterm-screen').boundingBox(); assert(box);
    await page.mouse.click(box.x + 40, box.y + 40);
    await page.keyboard.down('Shift'); await page.mouse.move(box.x + 20, box.y + 20); await page.mouse.down();
    await page.mouse.move(box.x + 180, box.y + 20); await page.mouse.up(); await page.keyboard.up('Shift');
    await page.mouse.wheel(0, -500); await page.setViewportSize({width: 1000, height: 800}); await page.setViewportSize({width: 1920, height: 1200});
    // Exact reattachment while the real CLI is still running. No CLI input replay.
    await page.reload(); await page.locator('#sodaspaces-data[aria-busy=false]').waitFor();
    await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor(); await screen.focus();
    checkActive(); await page.keyboard.press('Control+c'); await page.keyboard.press('Control+c');
    // Ordinary SSH PTY comparison uses personal SSH credentials already configured
    // in the restricted file. Prompt travels on stdin, never argv or evidence.
    const alias = project.ssh[actor]; assert(alias);
    const args = sshArgs(request, project, actor);
    const ssh = Bun.spawn(['ssh', '-tt', ...args, 'exec ' + scenario.tool], {env: {...process.env, TERM: 'xterm-256color'}, stdin: 'pipe', stdout: 'pipe', stderr: 'pipe'});
    let wire = '', chunks = 0, outputBytes = 0; const decoder = new TextDecoder();
    const output = (async () => {for await (const bytes of ssh.stdout) {wire = (wire + decoder.decode(bytes, {stream: true})).slice(-65536); chunks++; outputBytes += bytes.length;}})();
    const errors = (async () => {for await (const _ of ssh.stderr) { /* Never retain private diagnostics. */ }})();
    try {
      deadline = Date.now() + 15000;
      while (!wire.includes(scenario.ready_text) && Date.now() < deadline) {checkActive(); await new Promise(resolve => setTimeout(resolve, 100));}
      assert(wire.includes(scenario.ready_text), 'Declared SSH CLI readiness not observed; no prompt submitted');
      checkActive(); ssh.stdin.write(prompt + '\n'); ssh.stdin.flush();
      deadline = Date.now() + 90000;
      while (!wire.includes(scenario.expected_text) && Date.now() < deadline) {checkActive(); await new Promise(resolve => setTimeout(resolve, 250));}
      const comparison = cliProtocolObservation(wire, scenario.expected_text); assert(comparison.expected_output, 'SSH CLI fixture output not confirmed');
      assert(outputBytes >= scenario.minimum_output_bytes && chunks > 1, 'Declared SSH streaming volume not observed');
      results.push({tool: scenario.tool, version: scenario.version, native_context: nativeContext, browser_version: page.context().browser()?.version(), browser, browser_stream: browserStream, ssh: comparison, ssh_chunks: chunks, ssh_bytes: outputBytes,
        reconnect: 'exact managed locator; no input replay', interrupt: 'Ctrl-C sent',
        review_required: ['Unicode/cursor correctness', 'alternate-screen redraw', 'mouse/paste semantics', 'history/selection', 'resize/interrupt', 'streaming', 'ordinary SSH comparison'],
        outcome: 'observations only; not CLI acceptance'});
      ssh.stdin.write('\x03\x03'); ssh.stdin.flush();
    } finally {
      ssh.stdin.end(); ssh.kill(); await ssh.exited; await Promise.all([output, errors]); wire = ''; stream.clear(session.id);
    }
  }
  return results;
}
