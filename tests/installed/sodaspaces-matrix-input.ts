// Additional, exact approval. The old single-terminal flag/input cannot select this journey.
import assert from 'node:assert/strict';
import {object, validID} from './sodaspaces-input';
import type {JourneyInput} from './sodaspaces-input';
export interface MatrixProject {environment: string; repository_id: string; repository_path: string; ssh: [string, string]}
export interface CLIScenario {tool: 'codex' | 'claude' | 'pi'; version: string; prompt_file: string; expected_text: string; ready_text: string; minimum_output_bytes: number}
export interface MatrixInput {
  target: string; revision: string; actors: [string, string]; sessions_per_actor: 6;
  actions: ['create', 'attach', 'hide', 'return', 'end']; ssh_config: string;
  projects: [MatrixProject, MatrixProject]; cli: CLIScenario[];
  provider_use: 'none' | 'browser-and-ssh-for-declared-clis';
  cli_effects: string[];
}
export function matrixInput(value: unknown, base: JourneyInput): MatrixInput {
  const input = object(value);
  assert.deepEqual(Object.keys(input).sort(), ['actions', 'actors', 'cli', 'cli_effects', 'projects', 'provider_use', 'revision', 'sessions_per_actor', 'ssh_config', 'target']);
  assert.equal(input.target, base.target); assert.equal(input.revision, base.revision);
  assert.deepEqual(input.actors, base.users.map(user => user.id));
  assert.equal(input.sessions_per_actor, 6);
  assert.deepEqual(input.actions, ['create', 'attach', 'hide', 'return', 'end']);
  assert(typeof input.ssh_config === 'string' && input.ssh_config.startsWith('/'));
  assert(Array.isArray(input.projects) && input.projects.length === 2);
  const projects = input.projects.map((value): MatrixProject => {
    const project = object(value);
    assert.deepEqual(Object.keys(project).sort(), ['environment', 'repository_id', 'repository_path', 'ssh']);
    const {environment, repository_id, repository_path, ssh} = project;
    assert(typeof environment === 'string' && /^p[0-9a-f]{24}$/.test(environment)); assert(validID(repository_id));
    assert(typeof repository_path === 'string' && /^\/[A-Za-z0-9][A-Za-z0-9_.-]*\/[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(repository_path));
    assert(Array.isArray(ssh) && ssh.length === 2 && ssh.every(alias => typeof alias === 'string' && /^soda-matrix-[a-z0-9-]{1,50}$/.test(alias)));
    const [a, b] = ssh; assert(typeof a === 'string' && typeof b === 'string' && a !== b);
    return {environment, repository_id, repository_path, ssh: [a, b]};
  });
  const [first, second] = projects;
  assert(first && second && first.environment !== second.environment && first.repository_id !== second.repository_id && first.repository_path !== second.repository_path);
  assert.equal(first.repository_id, base.repository_id); assert.equal(first.repository_path, base.repository_path);
  assert(new Set(projects.flatMap(project => project.ssh)).size === 4);
  assert(Array.isArray(input.cli) && (input.cli.length === 0 || input.cli.length === 3));
  assert.equal(input.provider_use, input.cli.length ? 'browser-and-ssh-for-declared-clis' : 'none');
  const cli_effects = input.cli.length ? ['personal-cli-state', 'provider-calls', 'browser-and-ssh-pty', 'interactive-input', 'interrupt-and-disconnect'] : [];
  assert.deepEqual(input.cli_effects, cli_effects);
  const cli = input.cli.map((value): CLIScenario => {
    const item = object(value); assert.deepEqual(Object.keys(item).sort(), ['expected_text', 'minimum_output_bytes', 'prompt_file', 'ready_text', 'tool', 'version']);
    const {tool, version, prompt_file, expected_text, ready_text, minimum_output_bytes} = item;
    assert(typeof ready_text === 'string' && ready_text.trim().length > 0 && ready_text.length <= 80 && !/[\r\n\0]/.test(ready_text) && ready_text !== expected_text);
    assert(typeof minimum_output_bytes === 'number' && Number.isInteger(minimum_output_bytes) && minimum_output_bytes >= 4096 && minimum_output_bytes <= 65536);
    assert(tool === 'codex' || tool === 'claude' || tool === 'pi');
    assert(typeof version === 'string' && version.length > 0 && version.length <= 160 && !/[\r\n\0]/.test(version));
    assert(typeof prompt_file === 'string' && prompt_file.startsWith('/'));
    assert(typeof expected_text === 'string' && expected_text.length > 0 && expected_text.length <= 80 && !/[\r\n\0]/.test(expected_text));
    return {tool, version, prompt_file, expected_text, ready_text, minimum_output_bytes};
  });
  assert.equal(new Set(cli.map(item => item.tool)).size, cli.length);
  return {target: base.target, revision: base.revision, actors: [base.users[0].id, base.users[1].id], sessions_per_actor: 6,
    actions: ['create', 'attach', 'hide', 'return', 'end'], ssh_config: input.ssh_config, projects: [first, second], cli,
    provider_use: cli.length ? 'browser-and-ssh-for-declared-clis' : 'none', cli_effects};
}
