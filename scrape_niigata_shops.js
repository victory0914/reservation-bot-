const { chromium } = require('playwright');
const fs = require('fs');

(async () => {
    const browser = await chromium.launch({ headless: true });
    const page = await browser.newPage();
    const url = 'https://www.cityheaven.net/niigata/soap/search/'; // SOAP category

    console.log(`Navigating to ${url}...`);
    await page.goto(url, { waitUntil: 'domcontentloaded' });

    // Wait for list to load
    await page.waitForTimeout(5000);

    // Extract all links
    const hyperlinks = await page.$$eval('a', as => as.map(a => a.href));

    // Filter for shop links
    // Often: https://www.cityheaven.net/niigata/A1501/A150101/shopname/
    const shopSet = new Set();
    hyperlinks.forEach(href => {
        // Matches .../niigata/AREA/SUBAREA/SHOPNAME/ (optional ending with / or query)
        const match = href.match(/cityheaven\.net\/niigata\/[A-Z0-9]+\/[A-Z0-9]+\/([^\/]+)\/?(?:$|\?)/);
        if (match && !href.includes('search')) {
            // Reconstruct clean URL
            // Extract full path part
            const fullMatch = href.match(/(https?:\/\/www\.cityheaven\.net\/niigata\/[A-Z0-9]+\/[A-Z0-9]+\/[^\/]+\/)/);
            if (fullMatch) {
                shopSet.add(fullMatch[1]);
            }
        }
    });

    const shops = Array.from(shopSet);
    console.log(`Found ${shops.length} unique shops.`);
    console.log(JSON.stringify(shops, null, 2));

    fs.writeFileSync('niigata_shops.json', JSON.stringify(shops, null, 2));

    await browser.close();
})();
