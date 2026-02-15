const { chromium } = require('playwright');
const fs = require('fs');

(async () => {
  const browser = await chromium.launch({ headless: false });
  const page = await browser.newPage();
  console.log("Navigating to the page...");
  await page.goto('https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/girlid-52809022/');
  console.log("Done navigating to the page.");
  await page.waitForTimeout(1000);
  console.log("Logging in");
  await page.click('a[href="#login"]');
  console.log("Clicked on log in button");
  await page.waitForTimeout(1000);
  console.log("Filling in the login form");
  await page.fill("input[type='text']", "amritacharya");
  await page.fill("input[type='password']", "12345678");
  console.log("Submitting the login form");
  await page.click('input[id="submitLogin"]');
  console.log("Logged in successfully");
  const cookies = await page.context().cookies();

  fs.writeFileSync('files/cookies.json', JSON.stringify(cookies, null, 2));
  console.log("Cookies saved to files/cookies.json");

  const user_agent = await page.evaluate(() => navigator.userAgent);
  fs.writeFileSync('files/user_agent.txt', user_agent);
  console.log("User agent saved to user_agent.txt");

  const localStorageData = await page.evaluate(() => {
    let data = {};
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      data[key] = localStorage.getItem(key);
    }
    return data;
  });

  fs.writeFileSync('files/localStorage.json', JSON.stringify(localStorageData, null, 2));

  console.log('LocalStorage saved');

  await page.waitForTimeout(5000);
  await browser.close();


})();