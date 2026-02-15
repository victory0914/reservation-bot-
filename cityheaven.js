// cityheaven.js
const { chromium } = require('playwright');
const readline = require('readline');

(async () => {
  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext();
  const page = await context.newPage();

  const girlId = 52800421;
  const girlUrl = `https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/girlid-${girlId}/`;

  // 1️⃣ Go to girl page
  console.log(`Navigating to girl page: ${girlUrl}`);
  await page.goto(girlUrl, { waitUntil: 'networkidle' });
  console.log('Page loaded.');

  // 2️⃣ Handle login if modal exists
  if (await page.locator('input[type="password"]').count() > 0) {
    console.log("Login modal detected. Starting login process...");

    await page.click('a[href="#login"]');
    console.log('Clicked login link. Waiting for login form...');
    await page.waitForSelector('#user', { state: 'visible' });
    console.log('Login form visible. Filling credentials...');

    await page.fill('#user', 'amritacharya');
    await page.fill('#pass', '12345678');
    console.log('Credentials filled. Submitting login form...');

    // Wait for AJAX login response instead of navigation
    await Promise.all([
      page.waitForResponse(resp =>
        resp.url().includes('login') && resp.status() === 200
      ),
      page.click('#submitLogin')
    ]);

    console.log("Login completed.");
  } else {
    console.log('No login modal detected. Proceeding without login.');
  }

  // 3️⃣ Click reserve button and wait for navigation
  //     console.log(await page.content());
  console.log("Clicking reserve button...");
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'networkidle' }),
    page.click('#reserve_btn')
  ]);

  

  // 4️⃣ Wait for calendar menu, allow manual intervention if not found
  console.log("Waiting for calendar menu (店舗の空き状況 tab)...");
  try {
    console.log('Looking for course selection tab (a#condition_course)...');
    await page.waitForSelector('a[id="condition_course"]', { timeout: 10000 });
    console.log('Found course selection tab. Clicking...');
    await page.click('a[id="condition_course"]');

    const course80 = page.locator('.recommend_table div:has-text("80分")');
    console.log('Waiting for 80分 course option to appear...');
    await course80.waitFor({ state: 'visible' });
    console.log('80分 course option visible. Clicking select button...');

    await course80
      .locator('button:has-text("選択する")')
      .click();
    console.log('Clicked select button for 80分 course.');

    console.log('Waiting for available slots to appear...');
    await page.waitForSelector('span[data-mark="o"]', { timeout: 20000 });

    const slots = page.locator('span[data-mark="o"]');
    const count = await slots.count();

    console.log("Available slots found:", count);

    if (count > 0) {
      await slots.first().click();
      console.log("Clicked first available slot");
    } else {
      console.log("No available slots.");
    }

  } catch (e) {
    console.log("Process failed at calendar/course/slot selection step.");
    console.log("Error details:", e);
    console.log("If you see the calendar tab, please click it manually, select the course and slot, then press Enter in the terminal to continue.");
    await new Promise(resolve => {
      const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
      rl.question('Press Enter after you have manually completed the calendar and slot selection, or want to finish...', () => {
        rl.close();
        resolve();
      });
    });
    return;
  }

  // 5️⃣ Wait for available slots
  console.log("Process completed. Waiting before closing...");
  await page.waitForTimeout(60000);

  console.log('Closing browser.');
  await browser.close();
  console.log('Browser closed. Script finished.');
})();
