const { chromium } = require('playwright');

(async () => {
    const browser = await chromium.launch({ headless: false });
    const page = await browser.newPage();

    const shops = ['arabiannight', 'esthetique', 'club-dia', 'honey', 'alice', 'premium', 'jewel'];
    const validShops = [];

    for (const shop of shops) {
        const url = `https://www.cityheaven.net/niigata/A1501/A150101/${shop}/A6ShopReservation/`;
        console.log(`Checking ${url}...`);
        try {
            const response = await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 5000 });
            if (response.status() === 200 && !page.url().includes('error')) {
                console.log(`VALID: ${shop}`);
                validShops.push(shop);
            } else {
                console.log(`INVALID: ${shop} (Status: ${response.status()}, URL: ${page.url()})`);
            }
        } catch (e) {
            console.log(`ERROR: ${shop} (${e.message})`);
        }
    }

    console.log("Valid shops:", validShops);
    await browser.close();
})();
