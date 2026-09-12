import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {test} from 'node:test';
import {chromium} from 'playwright';

// Uses the existing local screenshot session and a production-rendered row fixture.
// Generate it with SODA_FORGEJO_NOTIFICATION_GALLERY=1 go test ./scripts -run '^TestForgejoNotificationPreview'.
// No native notification status or subscription is changed.
test('notification page family remains readable in split desktop panes', {skip:process.env.SODA_FORGEJO_NOTIFICATION_REVIEW!=='1'},async()=>{
 const root=new URL('../../../',import.meta.url),origin='http://localhost:3300';
 const fixture=await readFile(new URL('.artifacts/forgejo-notification-refinement/populated.html',root),'utf8');
 const context=await chromium.launchPersistentContext(new URL('.local/screenshot-fixture-profile',root).pathname,{channel:'chrome',headless:true});
 const writes:string[]=[];
 await context.route(`${origin}/**`,r=>{if(!['GET','HEAD'].includes(r.request().method())){writes.push(r.request().method());return r.abort();}return r.continue();});
 try {
  const page=await context.newPage();
  for(const theme of ['light','dark']) for(const width of [320,640,800,960,1440]) {
   await page.setViewportSize({width,height:900});
   for(const path of ['/notifications','/notifications?q=read','/notifications/subscriptions','/notifications/watching']) {
    const response=await page.goto(origin+path,{waitUntil:'domcontentloaded'});
    assert.equal(response?.status(),200);assert.equal(page.url(),origin+path,'existing fixture login required');
    await page.evaluate(async theme=>{
     const link=document.querySelector<HTMLLinkElement>('link[href*="/assets/css/theme-"]')!;
     await new Promise<void>((resolve,reject)=>{link.onload=()=>resolve();link.onerror=()=>reject(new Error('Theme failed'));link.href=`/assets/css/theme-soda-${theme}.css`;});
     document.documentElement.dataset.theme=`soda-${theme}`;await document.fonts.ready;
    },theme);
    const expected=theme==='light'?'rgb(255, 255, 255)':'rgb(16, 16, 16)';
    assert.equal(await page.locator('body').evaluate(e=>getComputedStyle(e).backgroundColor),expected);
    assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),`${theme}/${width}/${path}`);
    const active=page.locator('.soda-tabs > .active').first();
    assert.equal(await active.evaluate(e=>getComputedStyle(e).color),expected);
    assert.equal(await active.evaluate(e=>getComputedStyle(e).textTransform),'uppercase');
    if(path==='/notifications') {
     // Fixture rows are rendered from notification_div.tmpl; page shell remains native.
     await page.locator('#notification_div').evaluate((el,html)=>el.outerHTML=html,fixture);
     assert.equal(await page.locator('.notifications-item').count(),5);
     assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
     for(const button of await page.locator('.notifications-buttons button').all()) {
      const b=await button.boundingBox();assert(b&&b.width>=44&&b.height>=44&&b.x>=0&&b.x+b.width<=width);
      assert.equal(await button.evaluate(el=>getComputedStyle(el).borderTopWidth),'1px');
     }
     const first=page.locator('.notifications-buttons button').first();await first.focus();
     assert.equal(await first.evaluate(el=>getComputedStyle(el).outlineStyle),'solid');
     const row=page.locator('.notifications-item').first();
     const cols=await row.evaluate(el=>getComputedStyle(el).gridTemplateColumns.split(' ').length);
     assert.equal(cols,width<1101?3:4);
     assert.equal(await page.locator('input[name="status"][value="read"]').count()>0,true);
    }
   }
  }
  assert.deepEqual(writes,[]);
 }finally{await context.close();}
});
