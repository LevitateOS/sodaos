import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {test} from 'node:test';
import {chromium} from 'playwright';

// Native Vue component with controlled GET results; no repositories or settings are changed.
test('dashboard repository browser supports both themes and terminal-sized panes', {skip:process.env.SODA_FORGEJO_SIDEBAR_REVIEW!=='1'},async()=>{
 const origin='http://localhost:3300';
 const context=await chromium.launchPersistentContext('.local/screenshot-fixture-profile',{channel:'chrome',headless:true});
 const writes:string[]=[];
 const records=['alice/activity-workbench','vince/activity-playground','vince/activity-field-notes-fork','bob/a-very-long-repository-name-that-must-stay-contained-in-a-narrow-pane'].map((name,i)=>({repository:{id:i+1,full_name:name,link:`/${name}`,fork:i===2,private:false,archived:false,mirror:false},latest_commit_status:null,locale_latest_commit_status:''}));
 await context.route(`${origin}/**`,async route=>{
  const request=route.request(),url=new URL(request.url());
  if(!['GET','HEAD'].includes(request.method())){writes.push(request.method());return route.abort();}
  if(url.pathname!=='/repo/search')return route.continue();
  const mode=url.searchParams.get('mode'),q=url.searchParams.get('q')??'';
  const data=records.filter(x=>x.repository.full_name.includes(q)&&(mode==='fork'?x.repository.fork:mode==='source'?!x.repository.fork:true));
  return route.fulfill({headers:{'x-total-count':String(data.length)},json:url.searchParams.has('count_only')?{ok:true}:{ok:true,data}});
 });
 try{
  const page=await context.newPage();
  await mkdir('.artifacts/forgejo-sidebar-refinement',{recursive:true});
  for(const theme of ['light','dark'])for(const width of [320,640,720,800,960,1199,1440]){
   await page.setViewportSize({width,height:1000});
   await page.goto(origin,{waitUntil:'domcontentloaded'});
   const rows=page.locator('.repo-owner-name-list > li > a');await rows.first().waitFor();
   await page.evaluate(async theme=>{
    const link=document.querySelector<HTMLLinkElement>('link[href*="/assets/css/theme-"]')!;
    await new Promise<void>((resolve,reject)=>{link.onload=()=>resolve();link.onerror=()=>reject(new Error('Theme failed'));link.href=`/assets/css/theme-soda-${theme}.css`;});
    document.documentElement.dataset.theme=`soda-${theme}`;await document.fonts.ready;
   },theme);
   // Native menu links transition color after a theme change.
   await page.waitForFunction(theme=>getComputedStyle(document.querySelector('#dashboard-repo-list .tabs-with-labels > .active')!).color===(theme==='light'?'rgb(223, 0, 27)':'rgb(255, 101, 117)'),theme);
   const sidebar=page.locator('.soda-dashboard-sidebar');
   assert.equal(await rows.count(),4);
   assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`${theme}/${width}: page overflow`);
   assert(await sidebar.evaluate(e=>e.scrollWidth<=e.clientWidth),`${theme}/${width}: sidebar overflow`);
   const active=page.locator('#dashboard-repo-list .tabs-with-labels > .active');
   assert.equal(await active.evaluate(e=>getComputedStyle(e).color),theme==='light'?'rgb(223, 0, 27)':'rgb(255, 101, 117)');
   for(const control of await page.locator('#dashboard-repo-list .tabs-with-labels > .item, #dashboard-repo-list input[type=search], #dashboard-repo-list .input > .dropdown, .repo-owner-name-list > li > a').all()){
    const box=await control.boundingBox();assert(box&&box.height>=44&&box.x>=0&&box.x+box.width<=width,`${theme}/${width}: target bounds`);
    assert.equal(await control.evaluate(e=>getComputedStyle(e).borderTopLeftRadius),'0px');
   }
   const feed=await page.locator('.soda-dashboard-feed').boundingBox(),side=await sidebar.boundingBox();
   assert(feed&&side);if(width<1200)assert(side.y>=feed.y+feed.height);else assert(side.x>=feed.x+feed.width);
   await sidebar.screenshot({path:`.artifacts/forgejo-sidebar-refinement/${theme}-${width}.png`});
   const dropdown=page.locator('#dashboard-repo-list .input > .dropdown');await dropdown.click();
   const menu=dropdown.locator(':scope > .menu');await menu.waitFor({state:'visible'});
   const box=await menu.boundingBox();assert(box&&box.x>=0&&box.x+box.width<=width);
   await page.keyboard.press('Escape');
   const search=page.locator('#dashboard-repo-list input[type=search]');await search.fill('activity-workbench');
   await page.waitForFunction(()=>document.querySelectorAll('.repo-owner-name-list > li').length===1);
   assert.match(await rows.first().innerText(),/activity-workbench/);
   await search.fill('');
   await page.waitForFunction(()=>document.querySelectorAll('.repo-owner-name-list > li').length===4);
   await page.locator('#dashboard-repo-list .repos-filter .item').filter({hasText:'Forks'}).click();
   await page.waitForFunction(()=>document.querySelectorAll('.repo-owner-name-list > li').length===1);
   assert.match(await rows.first().innerText(),/field-notes-fork/);
   const selectedFilter=page.locator('#dashboard-repo-list .repos-filter .active.item');
   assert.equal(await selectedFilter.evaluate(e=>getComputedStyle(e).borderRadius),'0px');
   assert.equal(await selectedFilter.evaluate(e=>getComputedStyle(e).backgroundColor),'rgba(0, 0, 0, 0)');
   const frame=page.locator('#dashboard-repo-list .dashboard-repos');
   assert.deepEqual(await frame.evaluate(e=>{const c=getComputedStyle(e);return [c.borderTopWidth,c.borderRightWidth,c.borderBottomWidth,c.borderLeftWidth];}),['1px','1px','1px','1px']);
   for(const el of await page.locator('#dashboard-repo-list .dashboard-repos > .attached.segment, #dashboard-repo-list .repo-owner-name-list').all()) {
    assert.deepEqual(await el.evaluate(e=>{const c=getComputedStyle(e);return [c.borderTopWidth,c.borderRightWidth,c.borderBottomWidth,c.borderLeftWidth];}),['0px','0px','0px','0px']);
   }
   await sidebar.screenshot({path:`.artifacts/forgejo-sidebar-refinement/simple-forks-${theme}-${width}.png`});
   await search.fill('nothing-matches');await page.waitForFunction(()=>document.querySelectorAll('.repo-owner-name-list > li').length===0);
   const tabs=page.locator('#dashboard-repo-list .tabs-with-labels > .item');await tabs.nth(1).click();
   assert.match(await tabs.nth(1).getAttribute('class')??'',/active/);
   await tabs.nth(0).click();assert(await search.isVisible());
  }
  assert.deepEqual(writes,[]);
 }finally{await context.close();}
});
