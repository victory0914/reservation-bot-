const { chromium } = require('playwright');
const fs = require('fs');

(async () => {
    const browser = await chromium.launch({
        headless: false,
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    });
    const context = await browser.newContext({
        userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36'
    });

    // Get girlId from command line arguments, default to a known ID for testing
    const girlId = process.argv[2] || '54844946';
    console.log(`Using Girl ID: ${girlId}`);

    // Load cookies if they exist
    if (fs.existsSync('files/cookies.json')) {
        try {
            const cookies = JSON.parse(fs.readFileSync('files/cookies.json', 'utf8'));
            await context.addCookies(cookies);
            console.log("Loaded cookies from files/cookies.json");
        } catch (e) {
            console.log("Failed to load cookies:", e.message);
        }
    }

    const page = await context.newPage();

    // Debug: Log console messages from the page
    page.on('console', msg => console.log('PAGE LOG:', msg.text()));
    page.on('pageerror', err => console.log('PAGE ERROR:', err.message));

    // Log requests
    page.on('request', request => console.log('>>', request.method(), request.url()));
    page.on('response', response => console.log('<<', response.status(), response.url()));

    try {
        const reservationUrl = `https://www.cityheaven.net/niigata/A1501/A150101/arabiannight/A6ShopReservation/?girl_id=${girlId}`;
        console.log("Navigating to reservation page:", reservationUrl);
        await page.goto(reservationUrl, { waitUntil: 'domcontentloaded' });

        console.log("Page loaded. Title: " + await page.title());
        console.log("Page URL: " + page.url());

        console.log("Looking for reservation frame or table...");

        // Check if we need to switch to an iframe (common in CityHeaven)
        let reservationFrame = page;
        try {
            const iframeElement = await page.waitForSelector('iframe#pcreserveiframe', { timeout: 15000 });
            if (iframeElement) {
                console.log("Found reservation iframe. Switching context...");
                reservationFrame = await iframeElement.contentFrame();
                if (!reservationFrame) {
                    console.log("Could not access iframe content frame, using page.");
                    reservationFrame = page;
                }
            }
        } catch (e) {
            console.log("No iframe found or timeout, checking main page...");
        }

        // Wait for the table to load in the correct context
        try {
            await reservationFrame.waitForSelector('table', { timeout: 20000 });
            console.log("Table found. checking availability...");
            const frameHtml = await reservationFrame.content();
            fs.writeFileSync('debug_frame_content.html', frameHtml);
            console.log("Frame content dumped to debug_frame_content.html");

            // Extract get_result JSON
            const resultMatch = frameHtml.match(/var get_result = '([^']+)';/);
            if (resultMatch && resultMatch[1]) {
                try {
                    const resultData = JSON.parse(resultMatch[1]);
                    console.log("Extracted get_result JSON data.");

                    // Check for any availability in the JSON
                    let jsonSlots = 0;
                    if (resultData.commu_acp_status) {
                        resultData.commu_acp_status.forEach(statusObj => {
                            for (const dateKey in statusObj) {
                                statusObj[dateKey].forEach(slot => {
                                    if (slot.acp_status_flg !== 'NOTGIRL' && slot.acp_status_flg !== 'ng') {
                                        // Check what flags denote availability. Usually NOTGIRL means not working.
                                        // Let's log unique flags we see to understand.
                                    }
                                    if (slot.acp_status_mark === '◎' || slot.acp_status_mark === '○' || slot.acp_status_mark === '△') {
                                        jsonSlots++;
                                        console.log(`JSON Slot found: ${JSON.stringify(slot)}`);
                                    }
                                });
                            }
                        });
                    }
                    console.log(`Total slots found in JSON data: ${jsonSlots}`);

                } catch (e) {
                    console.log("Error parsing get_result JSON:", e.message);
                }
            } else {
                console.log("Could not find get_result variable in frame content.");
            }

        } catch (e) {
            console.log("Error waiting for table:", e.message);
        }

        let slotFound = false;
        console.log("ARGV:", process.argv);
        // SCRAPING MODE
        if (process.argv.includes('scrape')) {
            console.log("Entering SCRAPE mode...");
            try {
                const shopTab = await reservationFrame.$('ul.tab-btn li.first-tab a');
                if (shopTab) {
                    console.log("Clicking Shop Availability...");
                    await Promise.all([
                        reservationFrame.waitForNavigation({ waitUntil: 'domcontentloaded' }),
                        shopTab.click()
                    ]);

                    // Wait for table
                    await reservationFrame.waitForSelector('table.cth', { timeout: 10000 });

                    // Scrape
                    const links = await reservationFrame.$$eval('a', as => as.map(a => a.href));
                    const girlIds = new Set();
                    links.forEach(link => {
                        const match = link.match(/girl(?:_id=|-)(\d+)/) || link.match(/\/(\d{7,})(\/|$)/);
                        if (match) girlIds.add(match[1]);
                    });

                    console.log(`Scraped ${girlIds.size} girl IDs.`);
                    fs.writeFileSync('girl_ids.json', JSON.stringify(Array.from(girlIds), null, 2));

                    await browser.close();
                    return;
                }
            } catch (e) {
                console.log("Scrape failed:", e.message);
            }
        }

        let maxWeeks = 4; // Check up to 4 weeks
        let currentWeek = 1;

        // Fetch calendar.js for debugging
        try {
            const calendarScript = await reservationFrame.evaluate(async () => {
                const response = await fetch('/js/calendar.js?202110141000'); // Use version if possible
                return await response.text();
            });
            fs.writeFileSync('debug_calendar.js', calendarScript);
            console.log("Dumped calendar.js to debug_calendar.js");
        } catch (e) {
            console.log("Error fetching calendar.js:", e.message);
        }

        while (!slotFound && currentWeek <= maxWeeks) {
            console.log(`Checking week ${currentWeek}...`);

            // Wait for table content
            try {
                await reservationFrame.waitForSelector('table.cth td', { timeout: 5000 });
            } catch (e) {
                console.log("Timeout waiting for table cells.");
            }

            // Look for slots in the frame
            // Verified Selector: td > span[data-mark="○"] (or ◎, △)
            const slotSelector = 'table.cth td > span[data-mark="◎"], table.cth td > span[data-mark="○"], table.cth td > span[data-mark="△"]';
            let availableSlot = await reservationFrame.$(slotSelector);

            if (availableSlot) {
                console.log("Available slot found in week " + currentWeek + ". Clicking...");
                await availableSlot.scrollIntoViewIfNeeded();
                await availableSlot.click();
                slotFound = true;

                // Wait for the frame to navigate or content to update
                console.log("Waiting for frame update...");
                await page.waitForTimeout(5000); // Wait for navigation/update

                // Re-acquire frame if needed or use existing if it persisted
                // DUMP FRAME CONTENT
                try {
                    const postClickHtml = await reservationFrame.content();
                    fs.writeFileSync('debug_post_click.html', postClickHtml);
                    console.log("Post-click frame content dumped to debug_post_click.html");
                } catch (e) {
                    console.log("Error dumping post-click frame:", e.message);
                }

                break; // Stop loop
            } else {
                console.log(`No slots found in week ${currentWeek}.`);
                // Check for next week button
                const nextWeekBtn = await reservationFrame.$('span.next-btn > a');
                if (nextWeekBtn) {
                    console.log("Navigating to next week...");
                    await Promise.all([
                        reservationFrame.waitForNavigation({ waitUntil: 'domcontentloaded' }),
                        nextWeekBtn.click()
                    ]);
                    currentWeek++;
                    // Re-assign frame if page navigated (for main page this is fine, for iframe might be tricky but variables should persist if frame object is valid, otherwise re-fetch)
                    if (reservationFrame !== page) {
                        try {
                            const iframeElement = await page.$('iframe#pcreserveiframe');
                            if (iframeElement) {
                                reservationFrame = await iframeElement.contentFrame();
                            }
                        } catch (e) { }
                    }
                } else {
                    console.log("No next week button found. Stopping.");
                    break;
                }
            }
        }



        if (slotFound) {
            console.log("Clicked slot. Waiting for course selection page...");
            await page.waitForTimeout(3000);

            // Re-acquire frame as the content likely changed (or we are in a new frame state)
            if (reservationFrame !== page) {
                try {
                    const iframeElement = await page.$('iframe#pcreserveiframe');
                    if (iframeElement) {
                        reservationFrame = await iframeElement.contentFrame();
                    }
                } catch (e) {
                    console.log("Lost frame context, trying page...");
                    reservationFrame = page;
                }
            }

            // Step 3: Course Selection (Cheapest)
            console.log("Analyzing courses for cheapest option...");
            try {
                // Wait for the CHOICE BUTTON to be visible (the price inputs are hidden)
                await reservationFrame.waitForSelector('input.choice-btn', { timeout: 10000 });

                // Evaluate to find the lowest price
                const cheapestPrice = await reservationFrame.evaluate(() => {
                    const inputs = Array.from(document.querySelectorAll('input[name="course_price"]'));
                    if (inputs.length === 0) return null;
                    const prices = inputs.map(input => parseInt(input.value, 10));
                    return Math.min(...prices);
                });

                if (cheapestPrice !== null) {
                    console.log(`Found cheapest price: ${cheapestPrice}`);
                    // Click the button associated with this price
                    // We select the form that has this price input, then the button inside it
                    const submitBtnSelector = `form:has(input[name="course_price"][value="${cheapestPrice}"]) input.choice-btn`;
                    const submitBtn = await reservationFrame.$(submitBtnSelector);
                    if (submitBtn) {
                        console.log("Submitting form for cheapest course (via form.submit())...");
                        console.log("Current URL before submit: " + page.url());

                        // Find the form element
                        const formSelector = `form:has(input[name="course_price"][value="${cheapestPrice}"])`;
                        const form = await reservationFrame.$(formSelector);

                        if (form) {
                            // Use Promise.all to wait for FRAME navigation, not page
                            console.log("Submitting form via CLICK...");
                            await Promise.all([
                                page.waitForTimeout(5000), // Give it time to submit/reload
                                submitBtn.click()
                            ]);
                            console.log("Form submitted via click (waited 5s).");
                        } else {
                            console.log("Form not found for submission.");
                        }

                        console.log("Current URL after submit (page): " + page.url());
                        try {
                            console.log("Current URL after submit (frame): " + reservationFrame.url());
                            const frameAfterHtml = await reservationFrame.content();
                            fs.writeFileSync('debug_iframe_after_submit.html', frameAfterHtml);
                            console.log("Dumped iframe content to debug_iframe_after_submit.html");
                        } catch (e) {
                            console.log("Could not dump frame content:", e.message);
                        }

                        console.log("Current URL after submit: " + page.url());
                    } else {
                        console.log("Could not find submit button for cheapest course.");
                    }
                } else {
                    console.log("No course prices found, searching for default button...");
                    const genericBtn = await reservationFrame.$('input.choice-btn');
                    if (genericBtn) await genericBtn.click();
                }

            } catch (e) {
                console.log("Course selection error or skipped:", e.message);
            }
        } else {
            console.log("No available slots found to proceed.");
            return;
        }

        // Step 4: Login
        console.log("Waiting for Login page...");
        await page.waitForTimeout(3000);

        // Login might be main page or iframe. Usually redirects to a login page.
        // Check if we are on login page
        // The header login is distinct from the reservation login form.
        // We look for input#user and input#pass in the main content.

        // We might need to switch frame again if it is still an iframe? 
        // Usually login is a full page redirect or inside the same iframe.
        // Let's check both contexts.

        let loginFrame = page;
        // Try checking iframe again just in case
        const iframeElement = await page.$('iframe#pcreserveiframe');
        if (iframeElement) {
            loginFrame = await iframeElement.contentFrame();
        }

        try {
            // Check visibility of login form
            // We want to avoid finding the hidden header login
            // The real login form usually has a visible submit button or distinct class
            const passInput = await loginFrame.$('input[name="pass"]');
            if (passInput) {
                const isVisible = await passInput.isVisible();
                if (isVisible) {
                    console.log("Visible login form detected.");
                    console.log("Entering credentials...");
                    await loginFrame.fill('input[name="user"]', 'amritacharya');
                    await loginFrame.fill('input[name="pass"]', '12345678');

                    const loginBtnSelectors = [
                        'div.loginButton',
                        'input#submitLogin',
                        'input.login',
                        'input[type="submit"]',
                        'button:has-text("ログイン")',
                        'input[value="ログイン"]'
                    ];

                    console.log("Taking screenshot before login click...");
                    await page.screenshot({ path: 'debug_before_login_click.png', fullPage: true });

                    let loginBtn = null;
                    for (const selector of loginBtnSelectors) {
                        loginBtn = await loginFrame.$(selector);
                        if (loginBtn && await loginBtn.isVisible()) {
                            console.log(`Found login button with selector: ${selector}`);
                            break;
                        }
                    }

                    if (loginBtn && await loginBtn.isVisible()) {
                        await loginBtn.click();
                        console.log("Login submitted.");
                        await page.waitForTimeout(5000);
                    } else {
                        console.log("Login button not found or not visible.");
                    }
                } else {
                    console.log("Login input found but NOT visible (likely header login). Navigation failed?");
                }
            } else {
                console.log("No login input found.");
            }

        } catch (e) {
            console.log("Login step error:", e.message);
        }

        // Step 5: Final Confirmation
        console.log("Reached final stage.");
        const finalHtml = await page.content();
        fs.writeFileSync('debug_final_stage.html', finalHtml);
        console.log("Final HTML dumped to debug_final_stage.html");

        console.log("Taking screenshot...");
        await page.screenshot({ path: 'reservation_final_stage.png', fullPage: true });

        // Save cookies
        const newCookies = await context.cookies();
        fs.writeFileSync('files/cookies.json', JSON.stringify(newCookies, null, 2));

        await browser.close();

    } catch (error) {
        console.error("An error occurred:", error);
        await page.screenshot({ path: 'error_state.png' });
        await browser.close();
    }
})();
