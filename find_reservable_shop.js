const { chromium } = require('playwright');

(async () => {
    const browser = await chromium.launch({ headless: true });
    const page = await browser.newPage();
    // Niigata list page
    const listUrl = 'https://www.cityheaven.net/niigata/soap/search/'; // Restricting to SOAP for consistency
    // Or just root niigata
    // Let's try to filter by "Net Reservation" if possible. 
    // Usually the list page has icons.

    console.log(`Navigating to ${listUrl}...`);
    await page.goto(listUrl, { waitUntil: 'domcontentloaded' });

    // Look for shops with "Net Reservation" icon or link
    // "ネット予約" (Net Yoyaku)
    // Selector might be tough.
    // We can look for links containing "A6ShopReservation".

    const reservationLinks = await page.$$eval('a[href*="A6ShopReservation"]', as => as.map(a => a.href));

    console.log(`Found ${reservationLinks.length} reservation links.`);

    if (reservationLinks.length > 0) {
        // Pick one unique shop URL
        const uniqueShops = new Set();
        for (const link of reservationLinks) {
            // Extract shop base URL part
            // .../shop/n/SHOPNAME/... or .../niigata/.../SHOPNAME/...
            // The reservation link is usually .../niigata/AREA/SUB/SHOP/A6ShopReservation/
            const match = link.match(/(https?:\/\/www\.cityheaven\.net\/[^\/]+\/[^\/]+\/[^\/]+\/[^\/]+\/)/);
            if (match) {
                uniqueShops.add(match[1] + "A6ShopReservation/");
            }
        }

        console.log("Unique reservable shops found:", Array.from(uniqueShops));
    } else {
        console.log("No direct reservation links found. Trying to parse shop list items.");
    }

    await browser.close();
})();
