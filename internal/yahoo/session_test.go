package yahoo

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewSession_GetsCookieAndCrumb(t *testing.T) {
	// Mock root server: returns A3 cookie
	root := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "abc123"})
		w.WriteHeader(http.StatusOK)
	}))
	defer root.Close()

	// Mock crumb server: returns crumb string
	crumb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test-crumb-xyz"))
	}))
	defer crumb.Close()

	sess, err := NewSession(root.URL, crumb.URL, "")
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if sess.Crumb() != "test-crumb-xyz" {
		t.Errorf("expected crumb 'test-crumb-xyz', got %q", sess.Crumb())
	}

	cookies := sess.Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "A3" && c.Value == "abc123" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected A3 cookie, got %v", cookies)
	}
}

func TestNewSession_Refresh(t *testing.T) {
	var callCount atomic.Int32

	root := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "refreshed"})
		w.WriteHeader(http.StatusOK)
	}))
	defer root.Close()

	crumb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte("crumb-v1"))
		} else {
			_, _ = w.Write([]byte("crumb-v2"))
		}
	}))
	defer crumb.Close()

	sess, err := NewSession(root.URL, crumb.URL, "")
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if sess.Crumb() != "crumb-v1" {
		t.Errorf("expected crumb 'crumb-v1', got %q", sess.Crumb())
	}

	if err := sess.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	if sess.Crumb() != "crumb-v2" {
		t.Errorf("expected crumb 'crumb-v2' after refresh, got %q", sess.Crumb())
	}
}

func TestNewSession_EUConsentFlow(t *testing.T) {
	// Consent server: serves form page and accepts POST
	consent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.SetCookie(w, &http.Cookie{Name: "A3", Value: "eu-cookie"})
			w.WriteHeader(http.StatusOK)
			return
		}
		// GET returns a page with sessionId and gcrumb in the URL-like content
		_, _ = w.Write([]byte(`<form><input name="sessionId" value="sess123"><input name="gcrumb" value="gc456"></form>`))
	}))
	defer consent.Close()

	// Root server: returns 302 to consent
	root := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", consent.URL+"/consent?sessionId=sess123&gcrumb=gc456")
		w.WriteHeader(http.StatusFound)
	}))
	defer root.Close()

	crumb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("eu-crumb"))
	}))
	defer crumb.Close()

	sess, err := NewSession(root.URL, crumb.URL, consent.URL)
	if err != nil {
		t.Fatalf("NewSession with EU consent failed: %v", err)
	}

	if sess.Crumb() != "eu-crumb" {
		t.Errorf("expected crumb 'eu-crumb', got %q", sess.Crumb())
	}

	found := false
	for _, c := range sess.Cookies() {
		if c.Name == "A3" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected A3 cookie from EU consent flow")
	}
}

func TestNewSession_CrumbSentWithCookies(t *testing.T) {
	root := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "val"})
		w.WriteHeader(http.StatusOK)
	}))
	defer root.Close()

	var gotCookie bool
	crumb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, c := range r.Cookies() {
			if c.Name == "A3" {
				gotCookie = true
			}
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer crumb.Close()

	_, err := NewSession(root.URL, crumb.URL, "")
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if !gotCookie {
		t.Error("crumb request should include cookies from root request")
	}
}
