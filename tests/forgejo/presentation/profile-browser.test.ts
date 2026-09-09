import assert from 'node:assert/strict';
import {test} from 'node:test';
import {chromium} from 'playwright';

// Read-only native profile checks; no follow, block or preference mutations.
test('public profile header and native tab layouts remain compact and usable', {skip:process.env.SODA_FORGEJO_LAYOUT_ORIGIN !== 'http://localhost:3300'}, async t => {
 const browser=await chromium.launch({channel:'chrome',headless:true});
 try {
  const page=await browser.newPage();
  for(const theme of ['light','dark']) for(const width of [1440,900,701,700,390,320]) {
   await t.test(`${theme} ${width}`,async()=>{
    await page.setViewportSize({width,height:width<500?844:1000});
    for(const route of ['/alice','/alice?tab=activity','/alice?tab=followers','/alice?tab=following','/alice?tab=stars','/alice/-/projects','/alice/-/packages']) {
     const response=await page.goto(`http://localhost:3300${route}`);
     assert(response); assert.equal(response.status(),200);assert.equal(page.url(),`http://localhost:3300${route}`);
     await page.evaluate(theme=>{document.documentElement.dataset.theme=`forgejo-${theme}`;document.documentElement.dataset.sodaLoginTheme=theme;},theme);
     assert.equal(await page.locator('.soda-profile-masthead').count(),1);
     assert.equal(await page.locator('#block-user').count(),1);
     const layout=await page.locator('.soda-profile-masthead').evaluate(e=>{const avatar=e.querySelector('img'); if(!avatar) throw Error('Missing profile avatar'); return {display:getComputedStyle(e).display,avatar:avatar.getBoundingClientRect().width,overflow:document.documentElement.scrollWidth>innerWidth};});
     assert.equal(layout.display,'grid');assert.equal(layout.avatar,width<=700?56:72);assert.equal(layout.overflow,false,route);
    }
    await page.goto('http://localhost:3300/alice');
    const menu=page.locator('.soda-profile-actions summary');
    await menu.focus();await page.keyboard.press('Enter');assert(await page.locator('.soda-profile-actions details').evaluate(e=>e instanceof HTMLDetailsElement && e.open));
    await page.keyboard.press('Escape');
   });
  }
 }finally{await browser.close();}
});
