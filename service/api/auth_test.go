package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBearerTokenFallsBackToHttpOnlyCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "cookie-token"})
	token, err := bearerToken(req)
	if err != nil {
		t.Fatal(err)
	}
	if token != "cookie-token" {
		t.Fatalf("got %q", token)
	}
}

func TestAuthorizationHeaderTakesPrecedenceOverCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/conversations", nil)
	req.Header.Set("Authorization", "Bearer header-token")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "cookie-token"})
	token, err := bearerToken(req)
	if err != nil {
		t.Fatal(err)
	}
	if token != "header-token" {
		t.Fatalf("got %q", token)
	}
}

func TestAuthRateLimiter(t *testing.T) {
	limiter := newRateLimiter()
	now := time.Now()
	for i := 0; i < authAttemptsPerWindow; i++ {
		if !limiter.allow("client", now) {
			t.Fatalf("attempt %d unexpectedly blocked", i+1)
		}
	}
	if limiter.allow("client", now) {
		t.Fatal("expected attempt to be blocked")
	}
	if !limiter.allow("client", now.Add(authWindow)) {
		t.Fatal("expected new window to allow request")
	}
}
