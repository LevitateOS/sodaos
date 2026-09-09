import assert from 'node:assert/strict';
import {test} from 'node:test';
import {mkdir} from 'node:fs/promises';
import {chromium} from 'playwright';

const enabled=process.env.SODA_FORGEJO_LAYOUT_ORIGIN==='http://localhost:3300';
const gallery=(theme: string)=>new URL(`../../../.artifacts/forgejo-presentation/repository-settings-${theme}.html`,import.meta.url).href;
const evidence=new URL('../../../.artifacts/repo-settings-overhaul/components/',import.meta.url);

// Production registry, navigation and runner partial, plus small native fixtures.
// These checks neither impersonate an administrator nor submit a native form.
test('repository settings compositions and navigation share the settings contract',{skip:!enabled},async t=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 try {
  const page=await browser.newPage();const errors: string[]=[];page.on('pageerror',error=>errors.push(error.message));
  await mkdir(evidence,{recursive:true});
  for(const theme of ['light','dark']) for(const width of [1440,1000,900,899,768,390,320]) {
   await t.test(`${theme} at ${width}px`,async()=>{
    await page.setViewportSize({width,height:width<500?844:1000});
    await page.goto(gallery(theme));await page.waitForSelector('.soda-settings-nav.is-enhanced');
    await page.evaluate(()=>document.fonts.ready);
    const layout=await page.evaluate(()=>{
     const container=document.querySelector('.soda-page-container');
     const content=document.querySelector('.repo-setting-content');
     const heading=document.querySelector('h1');
     if(!container || !content || !heading) throw Error('Missing settings layout markup');
     return {overflow:document.documentElement.scrollWidth>innerWidth,padding:getComputedStyle(container).paddingInlineStart,inset:getComputedStyle(content).paddingInlineStart,title:getComputedStyle(heading).fontSize,left:heading.getBoundingClientRect().left};
    });
    assert.deepEqual(layout,{overflow:false,padding:width<900?'16px':'24px',inset:'40px',title:'32px',left:width>1168?(width-1120)/2:width<900?16:24});
    for(const heading of await page.locator('.soda-settings-section > h2, .soda-form-section > legend').all()) {
     assert.equal(await heading.evaluate(el=>getComputedStyle(el).fontSize),'24px');
     assert.equal(await heading.evaluate(el=>getComputedStyle(el).marginInlineStart),'-40px');
    }
    for(const button of await page.locator('.repo-setting-content button').all()) {
     if(await button.isVisible()) {const bounds=await button.boundingBox(); assert(bounds); assert(bounds.height>=44,'shared button height');}
    }
    for(const message of await page.locator('.soda-notice').all()) assert(await message.isVisible());
    assert.equal(await page.locator('.soda-form-section').first().evaluate(el=>getComputedStyle(el).borderTopWidth),'0px');
    const nav=page.locator('.soda-settings-nav');
    const trigger=width<900?nav.locator('.soda-settings-current'):nav.locator('.soda-settings-nav-trigger').first();
    await trigger.focus();await page.keyboard.press('Enter');
    const general=nav.locator('a').first();assert(await general.isVisible());
    assert.equal(await general.getAttribute('aria-current'),'page');
    await general.focus();await page.keyboard.press('Escape');
    assert(await trigger.evaluate(el=>document.activeElement===el));assert(!(await general.isVisible()));
    await trigger.click();await page.locator('h1').click();assert(!(await general.isVisible()));
    if(width>=900) {
     await trigger.focus();await page.keyboard.press('ArrowDown');
     assert(await general.evaluate(el=>document.activeElement===el));
    } else {
     await trigger.click();
     assert(await nav.locator('a[href$="/actions/variables"]').isVisible());
     assert(await nav.locator('a[href$="/hooks/git"]').isVisible());
    }
    assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
    await page.keyboard.press('Escape');
    // Very long translated menu text remains inside the same menu canvas.
    await general.evaluate(el=>el.textContent='Eine außergewöhnlich lange lokalisierte Bezeichnung für Repository-Einstellungen');
    await trigger.click();assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
    await page.keyboard.press('Escape');
    if([1440,390,320].includes(width)) {
     await page.screenshot({path:new URL(`${theme}-${width}.png`,evidence).pathname,fullPage:true});
    }
   });
  }
  assert.deepEqual(errors,[]);
 } finally {await browser.close();}
});

test('repository settings navigation remains available without JavaScript',{skip:!enabled},async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 try {
  const page=await browser.newPage({javaScriptEnabled:false});
  for(const width of [1440,390,320]) {
   await page.setViewportSize({width,height:1000});await page.goto(gallery('dark'));
   const links=page.locator('.soda-settings-nav a');assert.equal(await links.count(),15);
   for(const link of await links.all()) assert(await link.isVisible());
   assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
  }
 } finally {await browser.close();}
});
