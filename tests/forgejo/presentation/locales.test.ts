import assert from 'node:assert/strict';
import {test} from 'node:test';
import {spawnSync} from 'node:child_process';
test('locale generation preserves native bytes and rejects namespace/duplicate-key collisions',()=>{
 const result=spawnSync('python3',['-c',`
import importlib.util
spec=importlib.util.spec_from_file_location('locales','scripts/forgejo-locales.py')
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
native='[common]\\nhome = Home %s\\n[settings]\\nprofile = Profile\\n'
extra='[soda]\\nnav_personal = Personal\\n'
assert module.merge(native,extra)==native+'\\n'+extra
for additions in ['[soda]\\nx = one\\nx = two\\n','[settings]\\nprofile = Wrong\\n']:
 try: module.merge(native,additions)
 except Exception: pass
 else: raise AssertionError('duplicate namespace/key accepted')
try: module.merge('[soda]\\nx = y\\n',extra)
except ValueError: pass
else: raise AssertionError('incomplete native catalog accepted')
`],{encoding:'utf8'});
 assert.equal(result.status,0,result.stderr);
});
