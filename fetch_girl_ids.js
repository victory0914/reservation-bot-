const { chromium } = require('playwright');
const fs = require('fs');

(async () => {
    const browser = await chromium.launch({ headless: true });
    const page = await browser.newPage();
    const reservationUrl = 'https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/A6ShopReservation/?girl_id=22859129';

    console.log(`Navigating to ${reservationUrl}...`);
    await page.goto(reservationUrl, { waitUntil: 'domcontentloaded' });

    // Wait for iframe
    const iframeElement = await page.waitForSelector('iframe#pcreserveiframe', { timeout: 15000 });
    const frame = await iframeElement.contentFrame();

    console.log("Frame loaded.");
    await frame.waitForLoadState('domcontentloaded');

    // Trying to click "Shop Availability"
    try {
        // Use more generic selector or text
        const shopTab = await frame.$('ul.tab-btn li.first-tab a');
        if (shopTab) {
            console.log("Found Shop Availability tab. Clicking...");
            await Promise.all([
                frame.waitForNavigation({ waitUntil: 'domcontentloaded' }),
                shopTab.click()
            ]);
            console.log("Navigated to Shop Availability.");
        } else {
            console.log("Shop availability tab not found.");
        }
    } catch (e) {
        console.log("Error clicking shop tab:", e.message);
    }

    // Wait for table
    try {
        await frame.waitForSelector('table.cth', { timeout: 5000 });
    } catch (e) {
        console.log("Table not found, maybe already there?");
    }

    // Dump all links
    const links = await frame.$$eval('a', as => as.map(a => a.href));
    const girlIds = new Set();

    links.forEach(link => {
        // Links like .../girlid-12345/ or .../girl_id=12345
        const match = link.match(/girl(?:_id=|-)(\d+)/) || link.match(/\/(\d{7,})(\/|$)/);
        if (match) {
            girlIds.add(match[1]);
        }
    });

    console.log(`Found ${girlIds.size} potential girl IDs:`, Array.from(girlIds));

    fs.writeFileSync('girl_ids.json', JSON.stringify(Array.from(girlIds), null, 2));

    await browser.close();
})();
