# City Heaven Low-Latency Client (Go)

This project is a high-performance, automated reservation client built in Go. It is designed to replace slower, browser-based automation (like Playwright/Puppeteer) by interacting directly with the HTTP API endpoints of the reservation system.

## Key Concepts

### 1. Direct HTTP Interactions vs. Browser Automation
- **Browser Automation (Playwright)**: Launches a full browser instance (Headless Chrome), loads CSS/JS, renders the page, and simulates clicks. This is easy to write but **slow** (seconds per action) and resource-heavy.
- **Direct HTTP (This Client)**: Reverses-engineers the network requests (API calls) made by the browser. It sends only the necessary data bytes. This is **extremely fast** (milliseconds per action) and lightweight.

### 2. Fingerprint Evasion (`uTLS`)
Websites often detect bots by analyzing the "TLS Handshake" (the initial secure connection setup). Standard Go HTTP clients (`net/http`) have a very distinct handshake that screams "I am a bot".
- **Solution**: We use the `uTLS` library.
- **How it works**: It mimics the TLS handshake of a real browser (e.g., Firefox or Chrome). To the server, the encrypted connection looks exactly like a human user's browser.

### 3. Session Management (Cookie Jar)
The client maintains a "Cookie Jar". Just like a browser, when the server sends a `Set-Cookie` header (e.g., after Login), the client stores it and automatically sends it back in subsequent requests. This keeps you logged in across the polling loop and reservation steps.

### 4. CSRF Handling
The site uses Cross-Site Request Forgery (CSRF) tokens to prevent unauthorized actions.
- **The Challenge**: The token changes on every page load and is hidden in the HTML (`<input name="_csrf" value="...">`).
- **The Solution**: Before making a `POST` request (e.g., "Select Course"), the client first performs a `GET` request to the page, downloads the HTML, and extracts the token using regex. It then includes this token in the `POST` data.

## Code Structure

### `client/`
- **`client.go`**: 
  - **`LowLatencyClient`**: A custom wrapper around Go's `http.Client`.
  - **`NewLowLatencyClient`**: Initializes the client with a `CookieJar` and the `uTLS` transport.
  - **`Do`**: Executes requests with automatic "User-Agent" injection.
- **`reservation.go`**: Contains the specific business logic for City Heaven.
  - **`FetchCalendar`**: Polls the availability table.
  - **`SelectSlot` / `SelectCourse` / `SubmitProfile`**: Methods that map to specific steps in the booking flow.

### `main.go`
- **Login**: Authenticates the user session.
- **Polling Loop**: Continuously checks the calendar (every 500ms by default) for an open slot.
- **Execution**: Once a slot is found, it immediately triggers the reservation sequence in step-by-step order.

## How to Run

1. **Clean & Build**:
   ```bash
   go mod tidy
   go build -o cityheaven_client
   ```

2. **Configure**:
   Open `main.go` and update the constants at the top:
   - `TargetShopID`, `TargetGirlID`, `TargetCourseID`
   - `Username`, `Password`
   - `DryRun` (Set to `true` to test without buying, `false` for real/live execution)

3. **Run**:
   ```bash
   ./cityheaven_client
   ```

## Safety Features
- **Dry Run**: Prevents the final "Buy" request from being sent during testing.
- **Rate Limiting**: The polling loop respects a configured interval (default 500ms) to avoid IP bans.
- **Panic Recovery**: Standard Go error handling ensures the app logs errors gracefully rather than crashing unexpectedly.
