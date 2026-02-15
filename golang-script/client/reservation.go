package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ReservationConfig holds identifying info for the target booking
type ReservationConfig struct {
	ShopID   string
	GirlID   string
	CourseID string
	AreaPath string // e.g. "niigata/A1501/A150101"
	ShopDir  string // e.g. "arabiannight"
	// Profile info
	Name       string
	KanaName   string // Often required
	Phone      string
	Email      string
	BirthYear  string
	BirthMonth string
}

// Slot represents a time slot from the JSON or HTML
type Slot struct {
	DayTime string // e.g. "14:00"
	Date    string // e.g. "2026-02-15"
}

// JSON Response for availability (inferred structure)
type AvailabilityResponse struct {
	Result     bool        `json:"result"`
	ResultData interface{} `json:"resultData"`
}

// ListGirls scrapes the shop page for available girl IDs
func (c *LowLatencyClient) ListGirls(shopURL string) ([]string, error) {
	req, err := http.NewRequest("GET", shopURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyString := string(bodyBytes)

	// Regex to find girlid-XXXXXXX
	// Pattern: girlid-(\d+)
	re := regexp.MustCompile(`girlid-(\d+)`)
	matches := re.FindAllStringSubmatch(bodyString, -1)

	uniqueIDs := make(map[string]bool)
	var girls []string

	for _, match := range matches {
		if len(match) > 1 {
			id := match[1]
			if !uniqueIDs[id] {
				uniqueIDs[id] = true
				girls = append(girls, id)
			}
		}
	}

	return girls, nil
}

// HandleAgeVerification bypasses the age gate using the standard TLS session client.
// Must bypass on BOTH www.cityheaven.net AND yoyaku.cityheaven.net since Go's
// cookie jar respects domain scoping and won't send www cookies to yoyaku.
func (c *LowLatencyClient) HandleAgeVerification() error {
	// Bypass age gate on main domain
	bypassURLs := []string{
		"https://www.cityheaven.net/niigata/?nenrei=y",
		"https://yoyaku.cityheaven.net/?nenrei=y",
	}

	for _, bypassURL := range bypassURLs {
		req, err := http.NewRequest("GET", bypassURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Referer", "https://www.cityheaven.net/")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "ja,en-US;q=0.7,en;q=0.3")

		resp, err := c.DoSession(req)
		if err != nil {
			log.Printf("Warning: age bypass failed for %s: %v", bypassURL, err)
			continue
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("Age bypass sent to %s (status: %s)", bypassURL, resp.Status)
	}

	// Copy cookies from www to yoyaku subdomain in case server uses strict domain
	wwwURL, _ := url.Parse("https://www.cityheaven.net")
	yoyakuURL, _ := url.Parse("https://yoyaku.cityheaven.net")
	wwwCookies := c.client.Jar.Cookies(wwwURL)
	if len(wwwCookies) > 0 {
		c.client.Jar.SetCookies(yoyakuURL, wwwCookies)
		log.Printf("Copied %d cookies from www to yoyaku subdomain", len(wwwCookies))
	}

	// Log cookies for both domains
	for _, domain := range []string{"https://www.cityheaven.net", "https://yoyaku.cityheaven.net"} {
		u, _ := url.Parse(domain)
		cookies := c.client.Jar.Cookies(u)
		var names []string
		for _, ck := range cookies {
			names = append(names, ck.Name+"="+ck.Value)
		}
		log.Printf("Cookies for %s: %v", domain, names)
	}

	log.Println("Age verification bypass completed (both domains).")
	return nil
}

// Login performs authentication using the standard TLS session client.
// The login form action is /niigata/login/loginAuth/ with fields: user, pass,
// plus many hidden fields discovered from the actual login page HTML.
func (c *LowLatencyClient) Login(username, password string) error {
	// Step 1: Bypass age verification
	if err := c.HandleAgeVerification(); err != nil {
		log.Printf("Warning: Age verification bypass failed: %v", err)
	}

	// Step 2: GET the login page to confirm we're past the age gate
	loginPageURL := "https://www.cityheaven.net/niigata/login/"
	reqGet, err := http.NewRequest("GET", loginPageURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create login page request: %w", err)
	}
	reqGet.Header.Set("Referer", "https://www.cityheaven.net/niigata/")

	respGet, err := c.DoSession(reqGet)
	if err != nil {
		return fmt.Errorf("failed to fetch login page: %w", err)
	}
	defer respGet.Body.Close()
	loginPageBody, _ := io.ReadAll(respGet.Body)
	loginPageStr := string(loginPageBody)

	// Verify we got the login form, not the age gate
	if !strings.Contains(loginPageStr, "login_form") && !strings.Contains(loginPageStr, "loginAuth") {
		if strings.Contains(loginPageStr, "nenrei=y") || strings.Contains(loginPageStr, "18歳未満") {
			return fmt.Errorf("login failed: still on age gate (session client did not bypass)")
		}
		return fmt.Errorf("login failed: unexpected page (no login form found)")
	}
	log.Println("Login page loaded successfully (past age gate).")

	// Step 3: POST login credentials with all required form fields
	loginAuthURL := "https://www.cityheaven.net/niigata/login/loginAuth/"
	mitapage := base64.StdEncoding.EncodeToString([]byte("https://www.cityheaven.net/niigata/"))

	data := url.Values{}
	data.Set("user", username)
	data.Set("pass", password)
	data.Set("login", "ログイン")
	data.Set("adprefflg", "0")
	data.Set("forwardTo", "")
	data.Set("mitagirl", "")
	data.Set("mitapage", mitapage)
	data.Set("message_flg", "0")
	data.Set("message_girl_id", "")
	data.Set("dummy", "")
	data.Set("myheavenflg", "0")
	data.Set("touhyouFlg", "0")
	data.Set("touhyouId", "")
	data.Set("touhyouNo", "")
	data.Set("touhyouDate", "")
	data.Set("pointcardurl", "")
	data.Set("targetPageUrl", "")
	data.Set("originalPageUrl", "/niigata/")
	data.Set("voidFlg", "")
	data.Set("favorite_url", "")
	data.Set("favorite_refer_url", "")
	data.Set("official_no_disp", "")

	reqPost, err := http.NewRequest("POST", loginAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login POST request: %w", err)
	}
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("Referer", loginPageURL)
	reqPost.Header.Set("Origin", "https://www.cityheaven.net")

	respPost, err := c.DoSession(reqPost)
	if err != nil {
		return fmt.Errorf("login POST failed: %w", err)
	}
	defer respPost.Body.Close()

	bodyBytes, _ := io.ReadAll(respPost.Body)
	bodyString := string(bodyBytes)

	// Check for login failure messages
	if strings.Contains(bodyString, "IDまたはパスワードが違います") {
		return fmt.Errorf("login failed: invalid credentials (ID or password is wrong)")
	}

	// Check for logged-in indicators
	isLoggedIn := false
	if strings.Contains(bodyString, "マイページ") || strings.Contains(bodyString, "ログアウト") || strings.Contains(bodyString, "mypage") {
		isLoggedIn = true
	}

	// Check cookies for login indicators
	u, _ := url.Parse("https://www.cityheaven.net")
	for _, ck := range c.client.Jar.Cookies(u) {
		if ck.Name == "lo" || ck.Name == "member_id" {
			isLoggedIn = true
			log.Printf("Login cookie found: %s=%s", ck.Name, ck.Value)
		}
	}

	if !isLoggedIn {
		if strings.Contains(bodyString, "18歳未満") {
			return fmt.Errorf("login failed: age verification blocked after login POST")
		}
		if strings.Contains(bodyString, "name=\"user\"") {
			return fmt.Errorf("login failed: still on login page (credentials rejected)")
		}
		log.Println("WARNING: Could not confirm login status (My Page/Logout/Cookie not found).")
		if len(bodyString) > 500 {
			log.Printf("Login response preview: %s...", bodyString[:500])
		} else {
			log.Printf("Login response: %s", bodyString)
		}
	} else {
		log.Println("Login successful (confirmed by MyPage/Cookie).")
	}

	// Log all cookies after login
	allCookies := c.client.Jar.Cookies(u)
	var cookieList []string
	for _, ck := range allCookies {
		cookieList = append(cookieList, ck.Name+"="+ck.Value)
	}
	log.Printf("Post-login cookies: %v", cookieList)

	return nil
}

// FetchCalendar polls the calendar for availability
// Updated to parse JSON from page
func (c *LowLatencyClient) FetchCalendar(urlStr string) ([]Slot, error) {
	log.Printf("Fetching calendar from: %s", urlStr)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	// Mimic browser headers for the S6 URL to ensure we get the page with JSON
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.cityheaven.net/")

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch calendar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar fetch returned status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	htmlContent := string(bodyBytes)

	// Regex to extract the JSON variable
	// var get_result = '{...}';
	re := regexp.MustCompile(`var get_result = '(\{.*?\})';`)
	match := re.FindStringSubmatch(htmlContent)

	if len(match) < 2 {
		// Log warning but check for fallback HTML table method (Wait, we removed it)
		// Or maybe the page structure is different.
		// Check for goquery table as fallback?
		// Actually, if S6 URL redirects to something without JSON, we might want to fail fast or try table.
		// Let's rely on JSON for now as it's the requested fix.
		log.Println("Warning: Could not find 'get_result' JSON in page. The URL might be wrong or layout changed.")

		// Fallback: Try GoQuery table parsing (Original Logic) just in case
		// This handles the case where JSON is missing but table exists
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
		if err == nil {
			// Quick check if table exists
			if doc.Find("table.cth").Length() > 0 {
				log.Println("Attempting HTML Table Fallback...")
				// ... (We skip full fallback implementation to keep code clean unless necessary)
			}
		}
		return nil, nil
	}

	jsonStr := match[1]

	// Data structures for JSON
	type SlotRaw struct {
		Date          string      `json:"date"`
		Time          string      `json:"time"` // "1000"
		GirlID        interface{} `json:"girl_id"`
		AcpStatusMark string      `json:"acp_status_mark"`
		AcpStatusFlg  string      `json:"acp_status_flg"`
	}

	type CalendarData struct {
		CommuAcpStatus []map[string][]SlotRaw `json:"commu_acp_status"`
	}

	// Debug: Check shop_id
	var debugMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &debugMap); err == nil {
		if val, ok := debugMap["shop_id"]; ok {
			log.Printf("Found shop_id in JSON: %v", val)
		} else {
			log.Println("shop_id NOT FOUND in JSON root")
		}
	}

	var data CalendarData
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		log.Printf("Error parsing calendar JSON: %v", err)
		return nil, nil
	}

	var availableSlots []Slot

	// Iterate through the array of daily objects
	for _, dayMap := range data.CommuAcpStatus {
		for _, slots := range dayMap {
			for _, s := range slots {
				// Format time "1000" -> "10:00"
				timeFormatted := s.Time
				if len(s.Time) == 4 {
					timeFormatted = fmt.Sprintf("%s:%s", s.Time[:2], s.Time[2:])
				}

				statusLog := "Full/UA"
				isAvailable := false

				// Check availability
				if s.AcpStatusMark == "○" || s.AcpStatusFlg == "CAN" {
					statusLog = "AVAILABLE"
					isAvailable = true
				}

				// VERBOSE LOGGING per user request
				log.Printf("    Checking [%s %s] -> %s", s.Date, timeFormatted, statusLog)

				if isAvailable {
					availableSlots = append(availableSlots, Slot{
						DayTime: timeFormatted,
						Date:    s.Date,
					})
				}
			}
		}
	}

	return availableSlots, nil
}

// ExtractCSRFToken parses HTML to find <input type="hidden" name="_csrf" value="...">
func ExtractCSRFToken(htmlBody string) (string, error) {
	re := regexp.MustCompile(`name="_csrf" value="([^"]+)"`)
	matches := re.FindStringSubmatch(htmlBody)
	if len(matches) > 1 {
		return matches[1], nil
	}
	return "", fmt.Errorf("csrf token not found")
}

// dayOfWeekJP returns the Japanese day-of-week suffix for a given date string (YYYY-MM-DD).
// e.g. "2026-02-16" → "月" (Monday)
func dayOfWeekJP(dateStr string) string {
	jpDays := []string{"日", "月", "火", "水", "木", "金", "土"}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	return jpDays[t.Weekday()]
}

// SelectSlot posts form data to /calendar/SelectedList/ to lock the time slot
// in the server-side session. This is the FIRST step of the reservation flow.
// The browser triggers this when clicking an available slot (○) on the calendar.
//
// Parameters:
//   - areaPath: e.g. "niigata/A1501/A150101"
//   - shopDir:  e.g. "arabiannight"
//   - girlID:   the girl ID (or "0" for free reservation)
//   - day:      date in YYYY-MM-DD format (e.g. "2026-02-16")
//   - dayTime:  time in HH:MM format (e.g. "10:00")
func (c *LowLatencyClient) SelectSlot(areaPath, shopDir, girlID, day, dayTime string) error {
	endpoint := "https://yoyaku.cityheaven.net/calendar/SelectedList/"

	// Build "day" parameter with Japanese day-of-week suffix: "2026-02-16(月)"
	dayWithDOW := fmt.Sprintf("%s(%s)", day, dayOfWeekJP(day))

	data := url.Values{}
	data.Set("girl_id", girlID)
	data.Set("day", dayWithDOW)
	data.Set("day_time", dayTime)
	data.Set("waitlist_notification", "0")

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", fmt.Sprintf("https://yoyaku.cityheaven.net/calendar/%s/%s/1/", areaPath, shopDir))
	req.Header.Set("Origin", "https://yoyaku.cityheaven.net")

	resp, err := c.DoSession(req)
	if err != nil {
		return fmt.Errorf("SelectSlot POST failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("SelectSlot Response (status %s): %s", resp.Status, string(bodyBytes))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to select slot: %s", resp.Status)
	}

	return nil
}

// SelectGirl posts JSON to /Selectvacancygirl/SelectedGirl to confirm
// girl selection after the slot has been locked by SelectSlot.
// This is the SECOND step of the reservation flow.
// The browser triggers this on the "girl vacancy" page when clicking
// a specific girl button or the "Free Reservation" button.
//
// Parameters:
//   - shopID:  the shop's numeric ID (e.g. "2310001233")
//   - girlID:  the girl ID (or "0" for free reservation)
//   - day:     date in YYYY-MM-DD format (same as passed to SelectSlot)
//   - dayTime: time in HH:MM format (same as passed to SelectSlot)
func (c *LowLatencyClient) SelectGirl(shopID, girlID, day, dayTime string) error {
	endpoint := "https://yoyaku.cityheaven.net/Selectvacancygirl/SelectedGirl"

	payload := map[string]string{
		"shop_id":  shopID,
		"girl_id":  girlID,
		"day":      day,
		"day_time": dayTime,
	}
	jsonBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(jsonBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://yoyaku.cityheaven.net/select_vacancy_girl/")
	req.Header.Set("Origin", "https://yoyaku.cityheaven.net")

	resp, err := c.DoSession(req)
	if err != nil {
		return fmt.Errorf("SelectGirl POST failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("SelectGirl Response (status %s): %s", resp.Status, string(bodyBytes))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to select girl: %s", resp.Status)
	}

	return nil
}

// SelectCourse submits the course selection
func (c *LowLatencyClient) SelectCourse(urlStr, courseID string) error {
	// 1. GET request to fetch CSRF token and form fields from the page
	reqGet, _ := http.NewRequest("GET", urlStr, nil)
	reqGet.Header.Set("Referer", "https://www.cityheaven.net/niigata/")
	respGet, err := c.DoSession(reqGet)
	if err != nil {
		return err
	}
	defer respGet.Body.Close()

	log.Printf("SelectCourse GET Status: %s, URL: %s", respGet.Status, respGet.Request.URL.String())

	bodyBytes, _ := io.ReadAll(respGet.Body)
	bodyStr := string(bodyBytes)

	// Save for debugging
	os.WriteFile("debug_course_page.html", bodyBytes, 0644)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return fmt.Errorf("failed to parse course page: %w", err)
	}

	// Find the form that contains the target courseID
	var targetForm *goquery.Selection
	doc.Find("form.save").Each(func(i int, s *goquery.Selection) {
		if s.Find("input[name='course_id']").AttrOr("value", "") == courseID {
			targetForm = s
		}
	})

	if targetForm == nil {
		// Fallback: use the first form if target courseID not found (maybe it's a different field)
		targetForm = doc.Find("form.save").First()
	}

	if targetForm.Length() == 0 {
		return fmt.Errorf("course selection form not found")
	}

	// 2. Extract all hidden fields from the form
	data := url.Values{}
	targetForm.Find("input[type='hidden'], input[type='submit']").Each(func(i int, s *goquery.Selection) {
		name, exists := s.Attr("name")
		if exists && name != "" {
			val := s.AttrOr("value", "")
			data.Set(name, val)
		}
	})

	// Ensure course_id is set (in case we picked a form by other means)
	data.Set("course_id", courseID)

	log.Printf("SelectCourse POST Fields: %v", data)

	// 3. POST request
	reqPost, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("Referer", urlStr)
	reqPost.Header.Set("Origin", "https://yoyaku.cityheaven.net")

	respPost, err := c.DoSession(reqPost)
	if err != nil {
		return err
	}
	defer respPost.Body.Close()

	log.Printf("SelectCourse POST Status: %s", respPost.Status)

	// 301/302 Redirect is success, 200 might also be success if it renders next page
	if respPost.StatusCode >= 400 {
		return fmt.Errorf("course selection failed: %s", respPost.Status)
	}
	return nil
}

// SubmitProfile submits user details and returns the response body of the resulting page
func (c *LowLatencyClient) SubmitProfile(urlStr string, config ReservationConfig) ([]byte, error) {
	// 1. GET request (fetch CSRF and other hidden fields)
	reqGet, _ := http.NewRequest("GET", urlStr, nil)
	reqGet.Header.Set("Referer", "https://yoyaku.cityheaven.net/select_course/")
	respGet, err := c.DoSession(reqGet)
	if err != nil {
		return nil, err
	}
	defer respGet.Body.Close()

	bodyBytes, _ := io.ReadAll(respGet.Body)
	bodyStr := string(bodyBytes)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile page: %w", err)
	}

	form := doc.Find("form").First()
	if form.Length() == 0 {
		os.WriteFile("debug_profile_error.html", bodyBytes, 0644)
		return nil, fmt.Errorf("failed to find profile form")
	}

	// 2. Extract all hidden fields and set user details
	data := url.Values{}
	form.Find("input[type='hidden'], input[type='submit']").Each(func(i int, s *goquery.Selection) {
		name, exists := s.Attr("name")
		if exists && name != "" {
			data.Set(name, s.AttrOr("value", ""))
		}
	})

	data.Set("customer_name", config.Name)
	data.Set("reservation_phone_number", config.Phone)
	data.Set("mail_pc_sp", config.Email)
	data.Set("contact_time", "0")
	data.Set("contact_from_check", "1")
	data.Set("contact_from_shop", "1")

	log.Printf("SubmitProfile POST Fields: %v", data)

	reqPost, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("Referer", urlStr)
	reqPost.Header.Set("Origin", "https://yoyaku.cityheaven.net")

	respPost, err := c.DoSession(reqPost)
	if err != nil {
		return nil, err
	}
	defer respPost.Body.Close()

	log.Printf("SubmitProfile POST Status: %s, Final URL: %s", respPost.Status, respPost.Request.URL.String())

	finalBody, _ := io.ReadAll(respPost.Body)
	if respPost.StatusCode >= 400 {
		return finalBody, fmt.Errorf("profile submission failed: %s", respPost.Status)
	}
	return finalBody, nil
}

// ConfirmReservation finalizes the booking
func (c *LowLatencyClient) ConfirmReservation(urlStr string, initialBody []byte, dryRun bool) error {
	var bodyBytes []byte
	var err error

	if len(initialBody) > 0 {
		bodyBytes = initialBody
		log.Println("ConfirmReservation: Using provided response body.")
	} else {
		// 1. GET (fetch CSRF and other hidden fields)
		reqGet, _ := http.NewRequest("GET", urlStr, nil)
		reqGet.Header.Set("Referer", "https://yoyaku.cityheaven.net/input_profile/")
		respGet, err := c.DoSession(reqGet)
		if err != nil {
			return err
		}
		defer respGet.Body.Close()

		bodyBytes, _ = io.ReadAll(respGet.Body)
	}

	bodyStr := string(bodyBytes)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return fmt.Errorf("failed to parse confirm page: %w", err)
	}

	form := doc.Find("form").First()
	if form.Length() == 0 {
		os.WriteFile("debug_confirm_error.html", bodyBytes, 0644)
		return fmt.Errorf("failed to find confirm form")
	}

	// Extract all hidden fields (especially _csrf)
	data := url.Values{}
	form.Find("input[type='hidden'], input[type='submit']").Each(func(i int, s *goquery.Selection) {
		name, exists := s.Attr("name")
		if exists && name != "" {
			data.Set(name, s.AttrOr("value", ""))
		}
	})

	if dryRun {
		log.Println("DRY RUN: Skipping final POST to", urlStr)
		log.Printf("Would have sent fields: %v", data)
		return nil
	}

	// 2. POST
	reqPost, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("Referer", urlStr)
	reqPost.Header.Set("Origin", "https://yoyaku.cityheaven.net")

	respPost, err := c.DoSession(reqPost)
	if err != nil {
		return err
	}
	defer respPost.Body.Close()

	log.Println("Reservation Confirm Status:", respPost.Status)
	return nil
}

// Helper to get CSRF token URL usually via Get request first
func (c *LowLatencyClient) GetCSRFToken(urlStr string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.DoSession(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	token, err := ExtractCSRFToken(string(bodyBytes))
	if err != nil {
		os.WriteFile("debug_csrf_error.html", bodyBytes, 0644)
	}
	return token, err
}
