package client

import (
	"log"
	"net/url"
)

// DebugCookies prints cookies for a given URL
func (c *LowLatencyClient) DebugCookies(urlStr string) {
	u, _ := url.Parse(urlStr)
	cookies := c.client.Jar.Cookies(u)
	log.Printf("Cookies for %s:", urlStr)
	for _, cookie := range cookies {
		log.Printf(" - %s = %s (Domain: %s)", cookie.Name, cookie.Value, cookie.Domain)
	}
}
