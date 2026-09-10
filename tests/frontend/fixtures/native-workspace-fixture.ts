// Test instrumentation only: reuse the existing workspace HTTP/socket model.
// Forgejo still supplies the complete native document, actor and drawer hooks.
import {installWorkspaceModel} from './workspace-fixture.js';
const actor = document.getElementById('soda-settings-link')?.dataset.actor;
const query = new URL(import.meta.url).searchParams;
const id = query.get('id'), owner = query.get('owner'), name = query.get('name');
if (!actor || !id || !owner || !name) throw Error('Missing native workspace fixture context');
window.nativeWorkspaceModel = installWorkspaceModel(actor, {id, owner, name});
declare global {interface Window {nativeWorkspaceModel: ReturnType<typeof installWorkspaceModel>}}
