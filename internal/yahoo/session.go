package yahoo

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"

// Session manages Yahoo Finance authentication (cookies + crumb).
type Session struct {
	client     *http.Client
	rootURL    string
	crumbURL   string
	consentURL string
	cookies    []*http.Cookie
	crumb      string
}

// NewSession creates a session by fetching cookies and crumb from Yahoo Finance.
func NewSession(rootURL, crumbURL, consentURL string) (*Session, error) {
	s := &Session{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 1 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		rootURL:    rootURL,
		crumbURL:   crumbURL,
		consentURL: consentURL,
	}
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	return s, nil
}

// Refresh re-fetches cookies and crumb.
func (s *Session) Refresh() error {
	if err := s.getCookie(); err != nil {
		return fmt.Errorf("get cookie: %w", err)
	}
	if err := s.getCrumb(); err != nil {
		return fmt.Errorf("get crumb: %w", err)
	}
	return nil
}

// Cookies returns the current session cookies.
func (s *Session) Cookies() []*http.Cookie { return s.cookies }

// Crumb returns the current crumb token.
func (s *Session) Crumb() string { return s.crumb }

func (s *Session) getCookie() error {
	req, err := http.NewRequest(http.MethodGet, s.rootURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check for EU consent redirect
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc := resp.Header.Get("Location")
		if strings.Contains(loc, "/consent") || strings.Contains(loc, "consent.yahoo") {
			return s.getCookieEU(resp)
		}
	}

	// Look for A3 cookie (fc.yahoo.com returns 404 with Set-Cookie, which is expected)
	cookies := parseSetCookieHeaders(resp.Header)
	if len(cookies) == 0 {
		// Fallback: try resp.Cookies() (works with httptest)
		cookies = resp.Cookies()
	}
	s.cookies = cookies

	hasA3 := false
	for _, c := range cookies {
		if c.Name == "A3" {
			hasA3 = true
		}
	}
	if !hasA3 {
		return errors.New("A3 cookie not found in response")
	}
	return nil
}

func (s *Session) getCookieEU(initialResp *http.Response) error {
	loc := initialResp.Header.Get("Location")

	// Collect cookies from the redirect chain
	var allCookies []*http.Cookie
	allCookies = append(allCookies, parseSetCookieHeaders(initialResp.Header)...)

	// Follow redirect to consent page (up to 3 hops)
	currentURL := loc
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(http.MethodGet, currentURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", userAgent)
		for _, c := range allCookies {
			req.AddCookie(c)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}
		allCookies = append(allCookies, parseSetCookieHeaders(resp.Header)...)

		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			currentURL = resp.Header.Get("Location")
			_ = resp.Body.Close()
			continue
		}

		_ = resp.Body.Close()
		break
	}

	// Extract sessionId and gcrumb from the consent URL
	parsed, err := url.Parse(currentURL)
	if err != nil {
		return fmt.Errorf("parse consent URL: %w", err)
	}
	sessionID := parsed.Query().Get("sessionId")
	gcrumb := parsed.Query().Get("gcrumb")

	// POST consent form
	form := url.Values{}
	form.Set("sessionId", sessionID)
	form.Set("gcrumb", gcrumb)
	form.Set("agree", "agree")

	postURL := currentURL
	if s.consentURL != "" {
		postURL = s.consentURL
	}

	req, err := http.NewRequest(http.MethodPost, postURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range allCookies {
		req.AddCookie(c)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	allCookies = append(allCookies, parseSetCookieHeaders(resp.Header)...)
	s.cookies = allCookies
	return nil
}

func (s *Session) getCrumb() error {
	crumbEndpoint := s.crumbURL + "/v1/test/getcrumb"
	// If crumbURL already contains the path, use it directly
	if strings.Contains(s.crumbURL, "/v1/test/getcrumb") {
		crumbEndpoint = s.crumbURL
	}

	req, err := http.NewRequest(http.MethodGet, crumbEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	for _, c := range s.cookies {
		req.AddCookie(c)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	crumb := strings.TrimSpace(string(body))
	if crumb == "" {
		return errors.New("empty crumb response")
	}
	s.crumb = crumb
	return nil
}

// parseSetCookieHeaders extracts cookies from Set-Cookie headers manually.
func parseSetCookieHeaders(h http.Header) []*http.Cookie {
	var cookies []*http.Cookie
	for _, line := range h.Values("Set-Cookie") {
		parts := strings.SplitN(line, ";", 2)
		if len(parts) == 0 {
			continue
		}
		nv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
		if len(nv) != 2 {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:  nv[0],
			Value: nv[1],
		})
	}
	return cookies
}
