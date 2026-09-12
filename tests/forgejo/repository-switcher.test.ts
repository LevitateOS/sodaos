import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {test} from 'node:test';
import {chromium} from 'playwright';

// Real Forgejo header and native dropdown bundle; controlled GET responses cover
// scopes and failures without creating accounts, memberships or repositories.
test('repository switcher reuses native dropdowns with scoped, cancellable search', {skip:process.env.SODA_FORGEJO_SWITCHER_REVIEW !== '1'}, async()=>{
 const origin='http://localhost:3300';
 const c=await chromium.launchPersistentContext('.local/screenshot-fixture-profile',{channel:'chrome',headless:true});
 try {
  await c.route(`${origin}/**`,r=>['GET','HEAD'].includes(r.request().method())?r.continue():r.abort());
  const p=await c.newPage();p.setDefaultTimeout(10000);const errors:string[]=[];p.on('pageerror',e=>errors.push(e.message));
  await p.goto(`${origin}/alice/activity-workbench`,{waitUntil:'load'});
  const root=p.locator('.soda-repository-breadcrumb');
  const actor=await root.getAttribute('data-user-id');assert(actor);
  const currentOwner=await root.getAttribute('data-owner-id');assert(currentOwner);
  const owner=p.locator('.soda-repository-switcher[data-kind=owner]'),repo=p.locator('.soda-repository-switcher[data-kind=repository]');
  const input=repo.locator('input');
  const requests:URL[]=[];
  let error=false, staleResolve:(()=>void)|undefined;
  await p.route('**/?soda-switcher-owners=1',r=>r.fulfill({contentType:'text/html',body:`<select data-soda-switcher-owners data-actor="${actor}"><option value="${actor}">soda-screenshot</option><option value="101">studio</option><option value="102">empty-org</option></select>`}));
  await p.route('**/repo/search?*',async r=>{
   const u=new URL(r.request().url());requests.push(u);
   if(error)return r.fulfill({status:500,body:'Server details must stay private'});
   const q=u.searchParams.get('q')??'';
   if(q==='stale')await new Promise<void>(resolve=>{staleResolve=resolve;});
   const count=q==='many'?15:q==='nothing'||u.searchParams.get('uid')==='102'?0:2;
   const ownerName=u.searchParams.get('uid')==='101'?'studio':'alice';
   const data=Array.from({length:count},(_,i)=>({repository:{full_name:i?`${ownerName}/${q||'private-fork'}`:`${ownerName}/activity-workbench`,link:i?`/${ownerName}/repo-${u.searchParams.get('page')}-${i}`:`/${ownerName}/activity-workbench`,private:i===1,fork:i===1}}));
   await r.fulfill({json:{ok:true,data}}).catch(()=>{});
  });
  await repo.click();await repo.locator('a.item').first().waitFor();
  assert.equal(requests.at(-1)?.searchParams.get('uid'),currentOwner);assert.equal(requests.at(-1)?.searchParams.get('exclusive'),'true');
  assert.equal(await input.evaluate(e=>document.activeElement===e),true);
  await input.fill('nothing');await repo.getByText('No matching repositories.',{exact:true}).waitFor();
  await p.keyboard.press('Escape');await repo.locator('>.menu').waitFor({state:'hidden'});
  await repo.click();await repo.locator('a.item').first().waitFor();
  await input.fill('fresh');await repo.getByText('alice/fresh',{exact:true}).waitFor();
  await input.fill('stale');await p.waitForRequest('**/repo/search?*q=stale*');
  await input.fill('latest');await repo.getByText('alice/latest',{exact:true}).waitFor();staleResolve?.();
  assert.equal(await repo.getByText('alice/stale',{exact:true}).count(),0);
  error=true;await input.fill('failure');await repo.getByText('Could not load. Try again.',{exact:true}).waitFor();
  assert.equal(await repo.getByText('Server details must stay private').count(),0);
  error=false;await repo.getByText('Retry',{exact:true}).click();await repo.getByText('alice/failure',{exact:true}).waitFor();
  await input.fill('many');await repo.locator('.soda-switcher-more').waitFor();await repo.locator('.soda-switcher-more').click();
  await p.waitForFunction(()=>document.querySelectorAll('[data-kind="repository"] a.item').length===29);
  assert.equal(requests.at(-1)?.searchParams.get('page'),'2');
  await p.keyboard.press('Escape');await owner.click();await owner.getByText('studio',{exact:true}).waitFor();
  await owner.locator('input').fill('studio');await p.keyboard.press('ArrowDown');await p.keyboard.press('Enter');
  await repo.getByText('studio/activity-workbench',{exact:true}).waitFor();
  assert.equal(requests.at(-1)?.searchParams.get('uid'),'101');assert.equal(p.url(),`${origin}/alice/activity-workbench`);
  await p.keyboard.press('Escape');await owner.click();await owner.locator('input').fill('');await owner.getByText('All your repositories',{exact:true}).click();
  await repo.locator('a.item').first().waitFor();assert.equal(requests.at(-1)?.searchParams.get('uid'),actor);assert.equal(requests.at(-1)?.searchParams.get('exclusive'),'false');
  await p.keyboard.press('Escape');await owner.click();await owner.getByText('empty-org',{exact:true}).click();await repo.getByText('No matching repositories.',{exact:true}).waitFor();
  await p.keyboard.press('Escape');await owner.click();await owner.getByText('alice',{exact:true}).click();await repo.locator('a.item').first().waitFor();
  await mkdir('.artifacts/forgejo-repository-switcher',{recursive:true});
  for(const theme of ['light','dark'])for(const width of [320,640,720,800,960,1440]){
   await p.keyboard.press('Escape');await p.setViewportSize({width,height:900});
   await p.evaluate(async theme=>{
    const link=document.querySelector<HTMLLinkElement>('link[href*="/assets/css/theme-"]')!;
    if (!link.href.endsWith(`/assets/css/theme-soda-${theme}.css`)) await new Promise<void>((resolve,reject)=>{link.onload=()=>resolve();link.onerror=()=>reject();link.href=`/assets/css/theme-soda-${theme}.css`;});document.documentElement.dataset.theme=`soda-${theme}`;await document.fonts.ready;
   },theme);
   await repo.click();await repo.locator('a.item').first().waitFor();
   const box=await repo.locator('>.menu').boundingBox();assert(box&&box.x>=0&&box.x+box.width<=width&&box.y+box.height<=900,`${theme}/${width}`);
   assert.equal(await repo.locator('>.menu').evaluate(e=>getComputedStyle(e).borderRadius),'0px');
   assert.equal(await repo.locator('[aria-current=page]').count(),1);
   await p.screenshot({path:`.artifacts/forgejo-repository-switcher/${theme}-${width}.png`});
  }
  await input.focus();await p.keyboard.press('ArrowDown');
  const navigation=p.waitForRequest(r=>r.isNavigationRequest()&&r.url()===`${origin}/alice/activity-workbench`);
  await p.keyboard.press('Enter');await navigation;await p.waitForLoadState('load');
  assert.deepEqual(errors,[]);
  // Ordinary server links remain available without JavaScript.
  const guest=await c.browser()!.newContext({javaScriptEnabled:false});const g=await guest.newPage();
  await g.goto(`${origin}/alice/activity-workbench`,{waitUntil:'domcontentloaded'});
  assert.equal(await g.locator('.soda-repository-switcher').count(),0);
  assert.equal(await g.locator('.soda-repository-breadcrumb a').count(),2);await guest.close();
 }finally{await c.close();}
});
