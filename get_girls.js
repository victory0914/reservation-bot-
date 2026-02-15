const { chromium } = require('playwright');
(async () => {
    const browser = await chromium.launch({
        headless: false,
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    });
    const context = await browser.newContext({
        userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36'
    });
    const fs = require('fs');
    if (fs.existsSync('files/cookies.json')) {
        const cookies = JSON.parse(fs.readFileSync('files/cookies.json', 'utf8'));
        await context.addCookies(cookies);
    }
    const page = await context.newPage();
    await page.goto('https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/');
    console.log("Page URL:", page.url());
    await page.screenshot({ path: 'shop_page.png' });
    // fs is already required
    fs.writeFileSync('shop_page.html', await page.content());

    const girlIds = await page.evaluate(() => {
        const links = Array.from(document.querySelectorAll('a'));
        return links.map(link => {
            const href = link.getAttribute('href');
            if (!href) return null;
            const match = href.match(/girl_id=(\d+)/);
            return match ? match[1] : null;
        }).filter(id => id !== null);
    });
    console.log(JSON.stringify([...new Set(girlIds)]));
    await browser.close();
})();
