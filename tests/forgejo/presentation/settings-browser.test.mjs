import assert from 'node:assert/strict';
import {test} from 'node:test';
import {chromium} from 'playwright';
import {readFile} from 'node:fs/promises';
const origin=process.env.SODA_FORGEJO_LAYOUT_ORIGIN;
const enabled=origin==='http://localhost:3300';
// Separate opt-in fixture session avoids competing with screenshot captures.
test('personal settings navigation and native read-only journeys', {skip:!enabled || process.env.SODA_PERSONAL_SETTINGS_NATIVE!=='1'}, async t=>{
 const context=await chromium.launchPersistentContext('.local/screenshot-fixture-profile',{channel:'chrome',headless:true});
 try{
  context.setDefaultTimeout(5000); const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
  for(const width of [1440,1000,900,899,768,390,320]){
   await t.test(`navigation and task start at ${width}`,async()=>{
    await page.setViewportSize({width,height:width<500?844:1000});
    const response=await page.goto(origin+'/user/settings');assert.equal(response.status(),200);assert.equal(new URL(page.url()).pathname,'/user/settings');
    const nav=page.locator('.soda-settings-nav');
    await page.waitForSelector('.soda-settings-nav.is-enhanced');
    const trigger=width<900?nav.locator('.soda-settings-current'):nav.locator('.soda-settings-nav-trigger').first();
    await trigger.focus();await page.keyboard.press('Enter');
    const account=nav.locator('a[href$="/account"]');assert(await account.isVisible());
    await account.focus();await page.keyboard.press('Escape');assert(await trigger.evaluate(el=>document.activeElement===el));assert(!(await account.isVisible()));
    await trigger.click();await page.mouse.click(4,80);assert(!(await account.isVisible()));
    if(width<900){await trigger.click();assert(await nav.locator('a[href$="/applications"]').isVisible());assert(await nav.locator('a[href$="/repos"]').isVisible());await page.keyboard.press('Escape');}
    const bounds=await page.evaluate(()=>({w:innerWidth,scroll:document.documentElement.scrollWidth,title:document.querySelector('h1').getBoundingClientRect().top,task:document.querySelector('.soda-profile-portrait').getBoundingClientRect().top,left:document.querySelector('h1').getBoundingClientRect().left}));
    assert.equal(bounds.scroll,bounds.w);assert(bounds.title<300);assert(bounds.task<300);if(width<900)assert.equal(bounds.left,16);
    await page.locator('#avatar-settings summary').click();assert(await page.locator('#new-avatar').isVisible());
    assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
   });
  }
  await page.goto(origin+'/user/settings/account#password');await page.waitForSelector('[data-settings-editor][open]');assert(await page.locator('#old_password').isVisible());
  await page.goto(origin+'/user/settings/keys');await page.locator('#add-ssh-button').click();await page.waitForSelector('#add-ssh-key-panel:visible');
  await page.waitForFunction(()=>document.activeElement?.id==='ssh-key-title');
  await page.locator('#cancel-ssh-button').click();await page.waitForFunction(()=>document.activeElement?.id==='add-ssh-button');
  await page.goto(origin+'/user/settings/applications/tokens/new');
  for (const select of await page.locator('.access-token-select').all()) assert(await select.evaluate(el=>el.getBoundingClientRect().height>=44));
  await page.goto(origin+'/user/settings/blocked_users');
  assert.equal(await page.locator('.soda-blocked-users > .flex-item').first().evaluate(el=>getComputedStyle(el).backgroundColor),'rgba(0, 0, 0, 0)');
  await page.goto(origin+'/user/settings#avatar-settings');
  assert(await page.locator('[data-url$="/avatar/delete"]').evaluate(el=>el.getBoundingClientRect().height>=44));
  assert.deepEqual(errors,[]);
 }finally{await context.close();}
});
test('navigation and editor fallback remains expanded without JavaScript',{skip:!enabled},async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 try{
  const page=await browser.newPage({javaScriptEnabled:false,viewport:{width:390,height:844}});
  await page.goto(new URL('../../../.artifacts/forgejo-presentation/gallery-dark.html',import.meta.url).href);
  assert(await page.locator('.soda-settings-nav a[href$="/account"]').isVisible());
  assert(await page.locator('.soda-settings-nav a[href$="/applications"]').isVisible());
  assert(await page.locator('.soda-profile-portrait input[type=file]').isVisible());
 }finally{await browser.close();}
});

test('component error disclosures and long navigation labels remain accessible',{skip:!enabled},async()=>{
 const browser=await chromium.launch({channel:'chrome',headless:true});
 try{
  const page=await browser.newPage({viewport:{width:320,height:844}});
  const script=await readFile(new URL('../../../assets/branding/forgejo/personal-settings.js',import.meta.url),'utf8');
  await page.goto(new URL('../../../.artifacts/forgejo-presentation/gallery-dark.html',import.meta.url).href);
  // Isolated production fragment with a server-style error. No native POST.
  const html=await page.locator('#personal-settings-gallery').evaluate(el=>el.outerHTML);
  await page.evaluate(html=>{document.querySelector('#personal-settings-gallery').outerHTML=html;const root=document.querySelector('.soda-settings');const input=root.querySelector('input');input.closest('form').classList.add('field','error');root.querySelector('.soda-settings-nav a').textContent='Eine außergewöhnlich lange lokalisierte Bezeichnung für persönliche Einstellungen';},html);
  // A module scope lets the production enhancement initialize the new fixture.
  await page.addScriptTag({type:'module',content:script});
  await page.waitForSelector('[data-settings-editor][open]');
  assert(await page.locator('.soda-profile-portrait input').isVisible());
  await page.locator('.soda-settings-current').click();
  assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 }finally{await browser.close();}
});
