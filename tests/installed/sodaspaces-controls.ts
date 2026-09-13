// Current product UI paths shared by the installed journey and its local fixture.
// These helpers never authorize, fetch or substitute an API action.
import assert from 'node:assert/strict';
import type {Page} from 'playwright';
import {validID} from './sodaspaces-input.ts';
export async function projectView(page: Page, repository: string, view: 'Environment' | 'Access' = 'Environment') {
  assert(validID(repository));
  const controls = page.locator(`[data-project-controls][data-repository-id="${repository}"]`);
  if (!await controls.isVisible()) {
    await page.getByLabel('Workspace options', {exact: true}).click();
    await page.getByRole('button', {name: 'Repository environment / access', exact: true}).click();
  }
  await controls.locator(':scope[aria-busy=false]').waitFor();
  await controls.getByRole('tab', {name: view, exact: true}).click();
  return controls;
}
// Read/select the displayed original target before issuing a creation permit.
// The full page names directly created terminals; this helper never adds Rename.
export async function prepareManagedTerminal(page: Page, repositoryName: string, environment: string): Promise<string | undefined> {
  assert(/^p[0-9a-f]{24}$/.test(environment));
  if (await page.locator('#sodaspaces-data').getAttribute('data-workspace-kind') !== 'page') return undefined;
  const project = page.locator(`.soda-project-group[data-environment-id="${environment}"] .soda-project-select`);
  if (!await project.isVisible()) await page.getByRole('button', {name: 'Projects', exact: true}).click();
  assert.equal((await project.innerText()).trim(), repositoryName, 'The observed original project must match the selected target before submission');
  await project.click();
  const create = page.getByRole('button', {name: 'New terminal', exact: true});
  assert.equal(await create.getAttribute('data-environment-id'), environment);
  const name = await create.getAttribute('data-terminal-name'); assert(name && /^Terminal [1-9][0-9]*$/.test(name));
  return name;
}
export async function newManagedTerminal(page: Page, repositoryName: string, name: string, environment: string) {
  const directName = await prepareManagedTerminal(page, repositoryName, environment);
  if (directName !== undefined) {
    assert.equal(directName, name, 'Creation approval must use the displayed default name; no implicit Rename is permitted');
    await page.getByRole('button', {name: 'New terminal', exact: true}).click();
    await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
    return;
  }
  await page.getByRole('button', {name: 'New terminal', exact: true}).click();
  const chooser = page.getByRole('dialog', {name: 'New terminal', exact: true});
  assert(/^p[0-9a-f]{24}$/.test(environment));
  await chooser.getByLabel('Project', {exact: true}).selectOption({label: repositoryName});
  assert.equal(await chooser.getByLabel('Project', {exact: true}).inputValue(), environment, 'The observed original project must match the selected target before submission');
  await chooser.getByLabel('Terminal name', {exact: true}).fill(name);
  await chooser.getByRole('button', {name: 'Create terminal', exact: true}).click();
  await page.locator('.soda-workspace-terminal:visible .is-connected').waitFor();
}
export async function terminalMenu(page: Page, command: string) {
  const selected = page.locator('.soda-workspace-terminal:visible');
  assert.equal(await selected.count(), 1, 'This journey permits only its one selected terminal');
  await selected.getByLabel('Terminal actions', {exact: true}).click();
  await selected.getByRole('button', {name: command, exact: true}).click();
}
