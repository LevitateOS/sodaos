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
export async function newManagedTerminal(page: Page, repositoryName: string, name: string, environment: string) {
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
