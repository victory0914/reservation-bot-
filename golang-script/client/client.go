package client

import (
	"bufio"
	"bytes"
	"context"
	stdtls "crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

// RequestResult holds the timing and status of a request
type RequestResult struct {
	StartTime            time.Time     `json:"start_time"`
	DNSStart             time.Duration `json:"dns_start"`
	DNSDone              time.Duration `json:"dns_done"`
	ConnectStart         time.Duration `json:"connect_start"`
	ConnectDone          time.Duration `json:"connect_done"` // TCP Handshake complete
	TLSHandshakeStart    time.Duration `json:"tls_start"`
	TLSHandshakeDone     time.Duration `json:"tls_done"`
	WroteRequest         time.Duration `json:"wrote_request"` // Time when request was fully written
	GotFirstResponseByte time.Duration `json:"ttfb"`
	TotalDuration        time.Duration `json:"total_duration"`
	StatusCode           int           `json:"status_code"`
	Protocol             string        `json:"protocol"`
	ConnectionReused     bool          `json:"connection_reused"`
	Error                string        `json:"error,omitempty"`
	Body                 []byte        `json:"-"`
}

// LowLatencyClient wraps an http.Client with safety and timing features
type LowLatencyClient struct {
	client               *http.Client
	sessionClient        *http.Client // Standard TLS client for login/age-verification (shares cookie jar)
	mu                   sync.RWMutex
	shutdown             bool
	CancelGlobal         context.CancelFunc
	SimulateRemoteStatus int

	SafetyTriggeredAt time.Time
	SafetyReason      string

	ProxyManager       *ProxyManager
	FingerprintManager *FingerprintManager
	CaptchaSolver      CaptchaSolver
	SafetyManager      *SafetyManager
	Scheduler          *Scheduler

	ForceStandardTransport bool
}

func NewLowLatencyClient(cancel context.CancelFunc, simulateStatus int, pm *ProxyManager, fm *FingerprintManager, cs CaptchaSolver, forceStandard ...bool) *LowLatencyClient {
	// Create cookie jar for session management (shared between both clients)
	jar, _ := cookiejar.New(nil)

	useStandard := false
	if len(forceStandard) > 0 && forceStandard[0] {
		useStandard = true
	}

	var transport http.RoundTripper
	if useStandard {
		// Use standard http.Transport with Proxy
		transport = &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				if pm != nil {
					p := pm.GetNext()
					if p != "" {
						if parsed, err := url.Parse(p); err == nil {
							safeProxy := parsed.Host
							if parsed.User != nil {
								safeProxy = fmt.Sprintf("%s@%s", parsed.User.Username(), parsed.Host)
							}
							fmt.Printf("   🔄 [Main Proxy] %s %s → via %s://%s\n", req.Method, req.URL.Path, parsed.Scheme, safeProxy)
						}
						return url.Parse(p)
					}
				}
				fmt.Printf("   ⚠️  [Main Proxy] %s %s → NO PROXY (direct IP)\n", req.Method, req.URL.Path)
				return nil, nil
			},
			// 10G Optimization & H2 Support
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &stdtls.Config{
				NextProtos: []string{"h2", "http/1.1"},
			},
		}
	} else {
		transport = newFingerprintedTransport(pm)
	}

	return &LowLatencyClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
			Jar:       jar, // Enable cookie persistence
		},

		// Standard TLS client for age verification + login.
		// The server's age gate rejects the uTLS fingerprint, so we use
		// standard Go TLS for session establishment. Shares the same cookie jar
		// so cookies set during login are available to the uTLS client.
		// NOW ALSO ROUTES THROUGH SMARTPROXY for IP rotation.
		sessionClient: &http.Client{
			Timeout: 20 * time.Second,
			Jar:     jar,
			Transport: &http.Transport{
				Proxy: func(req *http.Request) (*url.URL, error) {
					if pm != nil {
						// Use sticky proxy: same IP for entire login/reservation flow
						// Redirects within a request chain MUST use the same IP
						p := pm.GetSticky()
						if p != "" {
							// Log proxy details (mask credentials)
							if parsed, err := url.Parse(p); err == nil {
								safeProxy := parsed.Host
								if parsed.User != nil {
									username := parsed.User.Username()
									safeProxy = fmt.Sprintf("%s@%s", username, parsed.Host)
								}
								log.Printf("   🔄 [Session Proxy] %s %s → via %s://%s (sticky)", req.Method, req.URL.Path, parsed.Scheme, safeProxy)
							}
							return url.Parse(p)
						}
					}
					log.Printf("   ⚠️  [Session Proxy] %s %s → NO PROXY (direct IP)", req.Method, req.URL.Path)
					return nil, nil
				},
				TLSHandshakeTimeout:   10 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		CancelGlobal:           cancel,
		SimulateRemoteStatus:   simulateStatus,
		ProxyManager:           pm,
		FingerprintManager:     fm,
		CaptchaSolver:          cs,
		SafetyManager:          NewSafetyManager(),
		Scheduler:              NewScheduler(),
		ForceStandardTransport: useStandard,
	}
}

func newFingerprintedTransport(pm *ProxyManager) http.RoundTripper {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Create a Transport that uses our custom DialTLSContext
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Get a proxy from the manager (rotate per connection)
			var proxyURL string
			if pm != nil {
				proxyURL = pm.GetNext()
				if proxyURL != "" {
					// Prepare "safe" URL for logging (hide credentials)
					safeURL := proxyURL
					if u, err := url.Parse(proxyURL); err == nil {
						if u.User != nil {
							u.User = url.User("******")
						}
						safeURL = u.String()
					}
					fmt.Printf("   🔄 [Proxy] Rotating to: %s\n", safeURL)
				}
			}

			// If proxy is set, dial via proxy
			if proxyURL != "" {
				return dialViaProxy(ctx, "tcp", addr, proxyURL)
			}
			return dialer.DialContext(ctx, network, addr)
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, _ := net.SplitHostPort(addr)

			// 1. TCP Connection (via proxy if configured)
			var conn net.Conn
			var err error

			// Get a proxy from the manager (rotate per connection)
			var proxyURL string
			if pm != nil {
				proxyURL = pm.GetNext()
				if proxyURL != "" {
					// Prepare "safe" URL for logging (hide credentials)
					safeURL := proxyURL
					if u, err := url.Parse(proxyURL); err == nil {
						if u.User != nil {
							u.User = url.User("******")
						}
						safeURL = u.String()
					}
					fmt.Printf("   🔄 [Proxy] Rotating to: %s\n", safeURL)
				}
			}

			if proxyURL != "" {
				conn, err = dialViaProxy(ctx, "tcp", addr, proxyURL)
				if err != nil {
					return nil, err
				}
			} else {
				conn, err = dialer.DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
			}

			// 2. uTLS Handshake
			// We MUST use HelloCustom and manually modify the spec to strictly enforce HTTP/1.1
			// The server is sending HTTP/2 frames which defaults http.Transport breaks on.
			uConn := utls.UClient(conn, &utls.Config{
				ServerName:         host,
				InsecureSkipVerify: true,
				NextProtos:         []string{"http/1.1"},
			}, utls.HelloCustom)

			// Get the base spec for Firefox
			spec, err := utls.UTLSIdToSpec(utls.HelloFirefox_Auto)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to get utls spec: %w", err)
			}

			// Edit ALPN extension to remove h2
			for i, ext := range spec.Extensions {
				if alpn, ok := ext.(*utls.ALPNExtension); ok {
					alpn.AlpnProtocols = []string{"http/1.1"}
					spec.Extensions[i] = alpn
				}
			}

			if err := uConn.ApplyPreset(&spec); err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to apply preset: %w", err)
			}

			err = uConn.Handshake()
			if err != nil {
				conn.Close()
				return nil, err
			}

			return uConn, nil
		},
		ForceAttemptHTTP2: false, // Strict H1.1
		MaxConnsPerHost:   1,     // Force frequent new connections to rotate proxies?
		// Or keep default. Let's Set MaxIdleConnsPerHost to 0 to disable keep-alive if we want strict rotation.
		// For now, let's keep it simple.
	}
}

// Do executes a request using the uTLS-fingerprinted client (for latency-critical requests)
func (c *LowLatencyClient) Do(req *http.Request) (*http.Response, error) {
	// 1. Safety Check
	if c.SafetyManager != nil && c.SafetyManager.IsTriggered() {
		return nil, fmt.Errorf("safety trigger active: %s", c.SafetyManager.TriggerReason)
	}

	// Inject default user agent if missing
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}

	resp, err := c.client.Do(req)

	// Log request result
	if err != nil {
		fmt.Printf("   ❌ [Do] %s %s → ERROR: %v\n", req.Method, req.URL.String(), err)
	} else {
		fmt.Printf("   ✅ [Do] %s %s → %s\n", req.Method, req.URL.String(), resp.Status)
	}

	// 2. Report Result to SafetyManager
	if c.SafetyManager != nil {
		if err != nil {
			c.SafetyManager.CheckError(err)
		} else {
			if !c.SafetyManager.CheckResponse(resp) {
				if c.CancelGlobal != nil {
					fmt.Println("   🛑 Safety Manager triggered global shutdown.")
					c.CancelGlobal()
				}
			}
		}
	}

	return resp, err
}

// DoSession executes a request using the standard TLS client (for login/age verification)
// All requests are routed through SmartProxy for IP rotation.
func (c *LowLatencyClient) DoSession(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")
	}
	start := time.Now()
	resp, err := c.sessionClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		log.Printf("   ❌ [DoSession] %s %s → ERROR after %v: %v", req.Method, req.URL.String(), duration, err)
		return resp, err
	}
	log.Printf("   ✅ [DoSession] %s %s → %s (%v)", req.Method, req.URL.String(), resp.Status, duration)
	return resp, err
}

func (c *LowLatencyClient) CookieJar() http.CookieJar {
	return c.client.Jar
}

func (c *LowLatencyClient) Shutdown(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.shutdown {
		c.shutdown = true
		c.SafetyTriggeredAt = time.Now()
		c.SafetyReason = reason
		fmt.Printf("\n[SAFETY KILL SWITCH] Stopping all operations. Reason: %s\n", reason)
		if c.CancelGlobal != nil {
			c.CancelGlobal()
		}
	}
}

func (c *LowLatencyClient) GetSafetyDetails() (bool, time.Time, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.shutdown, c.SafetyTriggeredAt, c.SafetyReason
}

func (c *LowLatencyClient) IsShutdown() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.shutdown
}

func (c *LowLatencyClient) ExecuteRequest(ctx context.Context, method, url string) (*RequestResult, error) {
	return c.ExecuteRequestWithBody(ctx, method, url, nil, "")
}

func (c *LowLatencyClient) ExecuteRequestWithBody(ctx context.Context, method, url string, body []byte, contentType string) (*RequestResult, error) {
	headers := make(map[string]string)
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	return c.ExecuteRequestWithHeaders(ctx, method, url, body, headers)
}

func (c *LowLatencyClient) ExecuteRequestWithHeaders(ctx context.Context, method, url string, body []byte, headers map[string]string) (*RequestResult, error) {
	if c.IsShutdown() {
		return nil, fmt.Errorf("global shutdown active")
	}

	var start time.Time
	var dnsStart, dnsDone, connStart, connDone, tlsStart, tlsDone, wroteReq, firstByte time.Time
	var reused bool

	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectStart:         func(_, _ string) { connStart = time.Now() },
		ConnectDone:          func(network, addr string, err error) { connDone = time.Now() },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(_ stdtls.ConnectionState, _ error) { tlsDone = time.Now() },
		WroteRequest:         func(_ httptrace.WroteRequestInfo) { wroteReq = time.Now() },
		GotFirstResponseByte: func() { firstByte = time.Now() },
		GotConn: func(info httptrace.GotConnInfo) {
			reused = info.Reused
		},
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Inject User Agent and Headers from FingerprintManager
	var ua string
	if c.FingerprintManager != nil {
		ua = c.FingerprintManager.GetRandomUserAgent()
		randomHeaders := c.FingerprintManager.GetRandomHeaders()
		for k, v := range randomHeaders {
			req.Header.Set(k, v)
		}
	} else {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	// Ensure User-Agent is set (randomized or default) overwrites if blank, or we can force it.
	// The original code only set it if missing. Let's force it if FM is present.
	if req.Header.Get("User-Agent") == "" || c.FingerprintManager != nil {
		req.Header.Set("User-Agent", ua)
	}

	// Apply custom headers (overwrites randomized ones if conflict)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Critical section: Execute the request
	start = time.Now()
	resp, err := c.client.Do(req)

	// CAPTCHA HANDLING
	// If response suggests CAPTCHA (e.g. 403 or specific content), try to solve.
	// For now, let's assume 403 *might* be CAPTCHA or Block.
	// In a real scenario, we'd read the body and check for "recaptcha" or similar.
	if err == nil && (resp.StatusCode == 403 || resp.StatusCode == 429) {
		// Simple heuristic: if 403, try to solve CAPTCHA if Solver is available.
		if c.CaptchaSolver != nil {
			fmt.Println("[CAPTCHA] Suspicious status code detected. Attempting to solve...")
			// Drain old body
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			// Solve
			// We need a site key. In real app, we'd scrape it from body.
			// Passing dummy site key for now.
			solution, solveErr := c.CaptchaSolver.Solve("DUMMY_SITE_KEY", url)
			if solveErr == nil {
				fmt.Printf("[CAPTCHA] Solved! Solution: %s. Retrying request...\n", solution)
				// Retry logic:
				// Re-create request (bodyReader is consumed, need to reset if possible)
				if body != nil {
					bodyReader = bytes.NewReader(body)
					req, _ = http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), method, url, bodyReader)
					// Verify headers again?
					req.Header.Set("User-Agent", ua)
				} else {
					req, _ = http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), method, url, nil)
					req.Header.Set("User-Agent", ua)
				}
				// Re-apply headers
				for k, v := range headers {
					req.Header.Set(k, v)
				}

				// Retry
				start = time.Now() // Reset start time for retry
				resp, err = c.client.Do(req)
			} else {
				fmt.Printf("[CAPTCHA] Failed to solve: %v\n", solveErr)
			}
		}
	}

	total := time.Since(start)

	result := &RequestResult{
		StartTime:         start,
		TotalDuration:     total,
		TLSHandshakeStart: time.Duration(0),
	}

	if !dnsStart.IsZero() {
		result.DNSStart = dnsStart.Sub(start)
	}
	if !dnsDone.IsZero() {
		result.DNSDone = dnsDone.Sub(start)
	}
	if !connStart.IsZero() {
		result.ConnectStart = connStart.Sub(start)
	}
	if !connDone.IsZero() {
		result.ConnectDone = connDone.Sub(start)
	}
	if !tlsStart.IsZero() {
		result.TLSHandshakeStart = tlsStart.Sub(start)
	}
	if !tlsDone.IsZero() {
		result.TLSHandshakeDone = tlsDone.Sub(start)
	}
	if !wroteReq.IsZero() {
		result.WroteRequest = wroteReq.Sub(start)
	}
	if !firstByte.IsZero() {
		result.GotFirstResponseByte = firstByte.Sub(start)
	}
	result.ConnectionReused = reused

	if err != nil {
		// Handle Safety Checking
		if c.SimulateRemoteStatus == 403 || (resp != nil && resp.StatusCode == 403) {
			go c.Shutdown("Received 403 Forbidden")
		}
		if c.SimulateRemoteStatus == 429 || (resp != nil && resp.StatusCode == 429) {
			go c.Shutdown("Received 429 Too Many Requests")
		}

		result.Error = err.Error()
		result.Body = []byte{}
		return result, nil // Return result with error info rather than skipping log
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		go c.Shutdown(fmt.Sprintf("HTTP %d Detected", resp.StatusCode))
	}

	result.StatusCode = resp.StatusCode
	result.Protocol = resp.Proto

	// Read Response Body
	bodyBytes, _ := io.ReadAll(resp.Body)
	result.Body = bodyBytes

	return result, nil
}

// dialViaProxy establishes a connection to the target address via the specified proxy.
// It supports both SOCKS5 and HTTP CONNECT tunneling.
func dialViaProxy(ctx context.Context, network, addr, proxyURL string) (net.Conn, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy url: %w", err)
	}

	switch u.Scheme {
	case "socks5":
		var auth *proxy.Auth
		if u.User != nil {
			password, _ := u.User.Password()
			auth = &proxy.Auth{
				User:     u.User.Username(),
				Password: password,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return dialer.Dial(network, addr)

	case "http", "https":
		// HTTP CONNECT Tunneling
		// 1. Dial TCP to the proxy server
		proxyDialer := &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		conn, err := proxyDialer.DialContext(ctx, "tcp", u.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to dial http proxy: %w", err)
		}

		// 2. Send CONNECT request
		// CONNECT target:port HTTP/1.1
		// Host: target:port
		// Proxy-Authorization: Basic <base64> (if needed)
		req := &http.Request{
			Method: "CONNECT",
			URL:    &url.URL{Host: addr},
			Host:   addr,
			Header: make(http.Header),
		}

		if u.User != nil {
			password, _ := u.User.Password()
			auth := u.User.Username() + ":" + password
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			req.Header.Set("Proxy-Authorization", basicAuth)
		}

		// Add standard headers that might be required by strict proxies
		// req.Header.Set("User-Agent", "Go-http-client/1.1")
		// req.Header.Set("Proxy-Connection", "Keep-Alive")

		if err := req.Write(conn); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to write connect request: %w", err)
		}

		// 3. Read response
		resp, err := http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read connect response: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			conn.Close()
			return nil, fmt.Errorf("proxy connect failed: %s", resp.Status)
		}

		return conn, nil

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
}
