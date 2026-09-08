import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {test} from 'node:test';
import {contracts} from './settings-contracts.mjs';
const root=new URL('../../../appliance/forgejo/templates/',import.meta.url);
test('personal settings preserve native controls and capability conditions through structural changes',async()=>{
 const snapshot=JSON.parse(await readFile(new URL('settings-native-contracts.json',import.meta.url)));
 for(const [path,native] of Object.entries(snapshot.entries)) {
  const actual=contracts(await readFile(new URL(path,root),'utf8'));
  for(const kind of ['controls','gates']) {
   const remaining=[...actual[kind]];
   for(const expected of native[kind]) {
    const index=remaining.indexOf(expected);
    assert(index>=0,`${path}: lost native ${kind}: ${expected}`);
    remaining.splice(index,1);
   }
  }
 }
});
test('profile and appearance retain independent native save boundaries',async()=>{
 const profile=await readFile(new URL('user/settings/profile.tmpl',root),'utf8');
 assert.equal((profile.match(/<form\b/g)||[]).length,2);
 assert(profile.indexOf('</form>')<profile.indexOf('<form',profile.indexOf('<form')+1),'avatar and identity are sibling forms');
 assert(profile.includes('list="pronouns"'));
 assert(!profile.includes(' autofocus'));
 const appearance=await readFile(new URL('user/settings/appearance.tmpl',root),'utf8');
 assert.equal((appearance.match(/<form\b/g)||[]).length,4);
 const nav=await readFile(new URL('user/settings/navbar.tmpl',root),'utf8');
 for(const gate of ['.HideNavbarLinks','.EnableActions','.EnablePackages','DisableWebhooks','.EnableQuota']) assert(nav.includes(gate));
 const links=[...nav.matchAll(/href="([^"]+)"/g)].map(m=>m[1]);assert.equal(links.length,new Set(links).size);
});
