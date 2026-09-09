import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {test} from 'node:test';
import {contracts} from './settings-contracts.ts';

const root=new URL('../../../appliance/forgejo/templates/',import.meta.url);
test('repository settings retain pinned native controls, gates and script hooks',async()=>{
 const snapshot=(await import('./repository-settings-native-contracts.json')).default;
 assert.equal(snapshot.upstream,'15.0.7');
 for(const [path,native] of Object.entries(snapshot.entries)) {
  const source=await readFile(new URL(path,root),'utf8');
  const actual=contracts(source);
  for(const kind of ['controls','gates'] as const) {
   const remaining=[...actual[kind]];
   for(const expected of native[kind]) {
    const index=remaining.indexOf(expected);
    assert(index>=0,`${path}: lost native ${kind}: ${expected}`);
    remaining.splice(index,1);
   }
   if(kind==='controls') assert.deepEqual(remaining,[],path+': unexpected submission control');
  }
  for(const hook of native.hooks) assert(source.includes(hook),path+': lost native script hook '+hook);
 }
});

test('repository settings preserve native footer and explicit form boundaries',async()=>{
 const head=await readFile(new URL('repo/settings/layout_head.tmpl',root),'utf8');
 const footer=await readFile(new URL('repo/settings/layout_footer.tmpl',root),'utf8');
 assert(head.includes('template "repo/header" .ctxData'));
 assert(head.includes('soda-settings-shell'));
 assert.equal((head.match(/<h1\b/g)||[]).length,1);
 assert(!head.includes('flex-container-nav'));
 assert(footer.includes('template "base/footer" .'));
 const units=await readFile(new URL('repo/settings/units.tmpl',root),'utf8');
 assert.equal((units.match(/<form\b/g)||[]).length,1);
 assert(units.includes('action="{{.RepoLink}}/settings/units"'));
 const options=await readFile(new URL('repo/settings/options.tmpl',root),'utf8');
 for(const modal of ['delete-repo-modal','transfer-repo-modal','archive-repo-modal']) {
  assert(options.includes(`id="${modal}"`),modal);
 }
});
