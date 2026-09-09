import {mountSodaspaces} from './sodaspaces-drawer.js';
import {id} from './sodaspaces-api.js';
const root = document.getElementById('spaces-page'), status = document.getElementById('spaces-page-status');
if (root && id(root.dataset.sodaActor)) {
  const workspace = mountSodaspaces(root, {kind: 'page', expectedUserId: root.dataset.sodaActor});
  void workspace.refresh(); if (status) status.hidden = true;
}
