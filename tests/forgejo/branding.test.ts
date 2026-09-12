import assert from 'node:assert/strict';
import {readFile,readdir} from 'node:fs/promises';
import {test} from 'node:test';
import payload from '../../internal/nativebuild/forgejo-payload.json';
const root=new URL('../../',import.meta.url);

test('Forgejo delivers no decorative robot artwork and retains the new identity',async()=>{
 const walk=async(dir:URL):Promise<string[]>=>{
  const files=await readdir(dir,{withFileTypes:true});
  return (await Promise.all(files.map(e=>e.isDirectory()?walk(new URL(e.name+'/',dir)):readFile(new URL(e.name,dir),'utf8').then(s=>[s])))).flat();
 };
 for(const source of await walk(new URL('appliance/forgejo/templates/',root))) assert(!source.includes('-papercraft.png'),'retired artwork remains in a production template');
 for(const [target,source] of Object.entries(payload)) assert(!target.includes('-papercraft.png')&&!source.includes('-papercraft.png'),'retired artwork remains in the staged payload');
 const light=await readFile(new URL('assets/branding/source/soda-symbol-brutalist.svg',root),'utf8');
 const dark=await readFile(new URL('assets/branding/source/soda-symbol-brutalist-dark.svg',root),'utf8');
 assert.equal(light.replace('fill="#101010"','fill="#ffffff"'),dark,'theme variants must have identical geometry and an open core');
 for(const source of [light,dark]) {assert(source.includes('fill="#df001b"'));assert.equal((source.match(/fill-rule="evenodd"/g)||[]).length,2);}
 const staging=await readFile(new URL('scripts/stage.py',root),'utf8');
 assert(staging.includes("'assets/branding/source/soda-symbol-brutalist.svg', images / name"));
});

test('shared neutral surfaces, links, actions and control edges retain contrast in both modes',async()=>{
 const css=await readFile(new URL('assets/branding/theme/palette.css',root),'utf8');
 const values=new Map([...css.matchAll(/--soda-([\w-]+):\s*(#[\da-f]{6});/gi)].map(m=>[m[1]!,m[2]!]));
 const luminance=(key:string)=>{
  const hex=values.get(key);assert(hex,key);
  const rgb=[1,3,5].map(i=>parseInt(hex.slice(i,i+2),16)/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4);
  return rgb[0]!*.2126+rgb[1]!*.7152+rgb[2]!*.0722;
 };
 const contrast=(a:string,b:string,min:number)=>{const x=luminance(a),y=luminance(b);assert((Math.max(x,y)+.05)/(Math.min(x,y)+.05)>=min,`${a} against ${b}`);};
 for(const mode of ['light','dark']) {
  for(const bg of ['canvas','surface','panel']) {
   for(const fg of ['text','muted','link','link-hover','link-pressed']) contrast(`${mode}-${fg}`,`${mode}-${bg}`,4.5);
   for(const fg of ['border','focus']) contrast(`${mode}-${fg}`,`${mode}-${bg}`,3);
  }
  for(const state of ['action','action-hover','action-pressed']) contrast(`${mode}-on-action`,`${mode}-${state}`,4.5);
 }
});

test('every stylesheet referenced by the Forgejo header is in the canonical payload',async()=>{
 const header=await readFile(new URL('appliance/forgejo/templates/custom/header.tmpl',root),'utf8');
 const entries=new Map(Object.entries(payload));
 for(const match of header.matchAll(/<link[^>]+href="([^"]+)"/g)) {
  const path=match[1]!.replace('{{AssetUrlPrefix}}','/assets').replace('{{AppSubUrl}}','').split('?')[0]!;
  if(!path.startsWith('/assets/'))continue;
  const source=entries.get('public'+path);assert(source,`missing stylesheet payload: ${path}`);
  const input=source.startsWith('@build/terminal-assets/')
   ? source.replace('@build/terminal-assets/','.artifacts/browser-terminal/vendor/') : source;
  assert(!input.startsWith('@build/'),`unresolved build stylesheet: ${source}`);
  assert((await readFile(new URL(input,root))).length>0,source);
 }
});
