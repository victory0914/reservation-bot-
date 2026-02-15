package client

import (
	"bytes"
	"context"
	stdtls "crypto/tls"
	"fmt"
	"io"
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

	ProxyURL string
}

func NewLowLatencyClient(cancel context.CancelFunc, simulateStatus int, proxyURL string) *LowLatencyClient {
	// Create cookie jar for session management (shared between both clients)
	jar, _ := cookiejar.New(nil)

	return &LowLatencyClient{
		client: &http.Client{
			Transport: newFingerprintedTransport(proxyURL),
			Timeout:   20 * time.Second,
			Jar:       jar, // Enable cookie persistence
		},
		// Standard TLS client for age verification + login.
		// The server's age gate rejects the uTLS fingerprint, so we use
		// standard Go TLS for session establishment. Shares the same cookie jar
		// so cookies set during login are available to the uTLS client.
		sessionClient: &http.Client{
			Timeout: 20 * time.Second,
			Jar:     jar,
		},
		CancelGlobal:         cancel,
		SimulateRemoteStatus: simulateStatus,
		ProxyURL:             proxyURL,
	}
}

func newFingerprintedTransport(proxyURL string) http.RoundTripper {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Create a Transport that uses our custom DialTLSContext
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// If proxy is set, use it for TCP dial
			if proxyURL != "" {
				u, err := url.Parse(proxyURL)
				if err != nil {
					return nil, err
				}
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
			}
			return dialer.DialContext(ctx, network, addr)
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, _ := net.SplitHostPort(addr)

			// 1. TCP Connection (via proxy if configured)
			var conn net.Conn
			var err error

			if proxyURL != "" {
				u, err := url.Parse(proxyURL)
				if err != nil {
					return nil, err
				}

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
				conn, err = dialer.Dial(network, addr)
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
	}
}

// Do executes a request using the uTLS-fingerprinted client (for latency-critical requests)
func (c *LowLatencyClient) Do(req *http.Request) (*http.Response, error) {
	// Inject default user agent if missing
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}
	return c.client.Do(req)
}

// DoSession executes a request using the standard TLS client (for login/age verification)
func (c *LowLatencyClient) DoSession(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")
	}
	return c.sessionClient.Do(req)
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Apply custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Critical section: Execute the request
	start = time.Now()
	resp, err := c.client.Do(req)
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
