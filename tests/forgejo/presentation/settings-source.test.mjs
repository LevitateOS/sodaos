import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
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
    // User-requested presentation removal: retain values only as hidden fields.
    if(path==='user/settings/profile.tmpl' && (expected.startsWith('input name="pronouns"') || expected.startsWith('input name="keep_pronouns_private"') || expected==='option value="{{.}}"' || expected==='{{range .CommonPronouns}}')) continue;
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
 assert(!profile.includes('list="pronouns"'));
 assert(profile.includes('<input type="hidden" name="pronouns" value="{{.SignedUser.Pronouns}}">'));
 assert(profile.includes('<input type="hidden" name="keep_pronouns_private" value="{{if .SignedUser.KeepPronounsPrivate}}true{{else}}false{{end}}">'));
 assert(!profile.includes('settings.pronouns'));
 assert(!profile.includes('settings.keep_pronouns_private'));
 assert(!profile.includes(' autofocus'));
 const appearance=await readFile(new URL('user/settings/appearance.tmpl',root),'utf8');
 assert.equal((appearance.match(/<form\b/g)||[]).length,4);
 const nav=await readFile(new URL('user/settings/navbar.tmpl',root),'utf8');
 for(const gate of ['.HideNavbarLinks','.EnableActions','.EnablePackages','DisableWebhooks','.EnableQuota']) assert(nav.includes(gate));
 const links=[...nav.matchAll(/href="([^"]+)"/g)].map(m=>m[1]);assert.equal(links.length,new Set(links).size);
});

test('public profiles remove only pronoun display from the native partial',async()=>{
 const actual=await readFile(new URL('shared/user/profile_big_avatar.tmpl',root),'utf8');
 assert(!actual.includes('GetPronouns'));
 const restored=actual.slice(actual.indexOf('\n')+1).replace('<span class="username">{{.ContextUser.Name}}</span>','<span class="username">{{.ContextUser.Name}} {{if .ContextUser.GetPronouns .IsSigned}} · {{.ContextUser.GetPronouns .IsSigned}}{{end}}</span>');
 assert.equal(createHash('sha256').update(restored).digest('hex'),actual.match(/upstream SHA-256 ([a-f0-9]{64})/)[1]);
 const admin=await readFile(new URL('admin/user/edit.tmpl',root),'utf8');
 assert(!admin.includes('settings.pronouns'));
 assert(admin.includes('<input type="hidden" name="pronouns" value="{{.User.Pronouns}}">'));
});
