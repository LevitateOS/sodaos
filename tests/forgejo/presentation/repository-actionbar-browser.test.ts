import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {test} from 'node:test';
import {chromium} from 'playwright';

test('native repository actions keep counts attached and fit split panes',{skip:process.env.SODA_FORGEJO_ACTIONBAR_REVIEW!=='1'},async()=>{
 const c=await chromium.launchPersistentContext('.local/screenshot-fixture-profile',{channel:'chrome',headless:true});
 const writes:string[]=[];
 await c.route('http://localhost:3300/**',r=>{if(!['GET','HEAD'].includes(r.request().method())){writes.push(r.request().method());return r.abort();}return r.continue();});
 try{
  await mkdir('.artifacts/forgejo-repo-actionbar',{recursive:true});const p=await c.newPage();
  for(const theme of ['light','dark'])for(const width of [320,640,720,800,960,1000,1100,1440]){
   await p.setViewportSize({width,height:1000});await p.goto('http://localhost:3300/alice/activity-workbench',{waitUntil:'domcontentloaded'});
   await p.locator('#sodaspaces-button').waitFor();
   await p.evaluate(async theme=>{const link=document.querySelector<HTMLLinkElement>('link[href*="/assets/css/theme-"]')!;await new Promise<void>((resolve,reject)=>{link.onload=()=>resolve();link.onerror=()=>reject();link.href=`/assets/css/theme-soda-${theme}.css`});document.documentElement.dataset.theme=`soda-${theme}`;await document.fonts.ready;},theme);
   const details=p.locator('.soda-repository-actions'),summary=details.locator(':scope > summary'),panel=details.locator('.soda-repository-action-menu');
   const compact=await summary.isVisible();assert.equal(compact,width<=1000);
   if(compact){assert.equal(await details.getAttribute('open'),null);await summary.click();}
   await panel.waitFor({state:'visible'});
   const box=await panel.boundingBox();assert(box&&box.x>=0&&box.x+box.width<=width,`${theme}/${width}: menu fits`);
   assert(await p.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
   const groups=panel.locator('.ui.labeled.button');assert.equal(await groups.count(),3);
   for(const group of await groups.all()){
    const action=group.locator(':scope > .ui.button'),count=group.locator(':scope > .ui.label');
    const a=await action.boundingBox(),b=await count.boundingBox();assert(a&&b);
    assert(a.height>=44&&b.height>=44&&b.width>=44);
    assert(Math.abs(a.x+a.width-b.x)<1&&Math.abs(a.y-b.y)<1&&Math.abs(a.height-b.height)<1,'action and count must share all edges');
    assert.deepEqual(await count.evaluate(e=>{const c=getComputedStyle(e);return [c.borderTopWidth,c.borderRightWidth,c.borderBottomWidth,c.borderLeftWidth,c.borderRadius];}),['1px','1px','1px','0px','0px']);
    assert(await count.getAttribute('href'),'count remains a native link');
    await action.hover();await count.focus();
   }
   const space=p.locator('#sodaspaces-button');assert.equal(await space.evaluate(e=>getComputedStyle(e).borderTopWidth),'1px');
   await p.locator('.repo-header').screenshot({path:`.artifacts/forgejo-repo-actionbar/${theme}-${width}.png`});
   if(compact){await panel.screenshot({path:`.artifacts/forgejo-repo-actionbar/menu-${theme}-${width}.png`});await p.keyboard.press('Escape');assert.equal(await details.getAttribute('open'),null);}
  }
  assert.deepEqual(writes,[]);
 }finally{await c.close();}
});
