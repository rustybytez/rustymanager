package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

const testToken = "test-secret"

func setupApp(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("AUTH_TOKEN", testToken)

	dsn := filepath.Join(t.TempDir(), "test.db")
	e, err := newApp(dsn)
	if err != nil {
		t.Fatalf("newApp: %v", err)
	}

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return srv
}

func TestUnauthenticatedRedirectsToLogin(t *testing.T) {
	srv := setupApp(t)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	for _, path := range []string{"/", "/projects", "/projects/new"} {
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("%s: got %d, want %d", path, resp.StatusCode, http.StatusSeeOther)
		}
		if loc := resp.Header.Get("Location"); loc != "/login" {
			t.Errorf("%s: redirect location = %q, want /login", path, loc)
		}
	}
}

func TestLoginPageRendered(t *testing.T) {
	srv := setupApp(t)

	resp, err := http.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /login: got %d, want 200", resp.StatusCode)
	}
}

func TestLoginWrongToken(t *testing.T) {
	srv := setupApp(t)

	form := url.Values{"token": {"wrong-token"}}
	resp, err := http.PostForm(srv.URL+"/login", form)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST /login wrong token: got %d, want 401", resp.StatusCode)
	}
}

func TestLoginCorrectTokenSetsCookie(t *testing.T) {
	srv := setupApp(t)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	form := url.Values{"token": {testToken}}
	resp, err := client.Post(srv.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("POST /login correct token: got %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/projects" {
		t.Errorf("redirect location = %q, want /projects", loc)
	}

	var authCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "auth_token" {
			authCookie = c
			break
		}
	}
	if authCookie == nil {
		t.Fatal("auth_token cookie not set")
	}
	if authCookie.Value != testToken {
		t.Errorf("cookie value = %q, want %q", authCookie.Value, testToken)
	}
}

func TestAuthenticatedAccessAllowed(t *testing.T) {
	srv := setupApp(t)

	jar := &singleCookieJar{name: "auth_token", value: testToken, host: srv.URL}
	client := &http.Client{Jar: jar}

	resp, err := client.Get(srv.URL + "/projects")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /projects authenticated: got %d, want 200", resp.StatusCode)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	srv := setupApp(t)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	resp, err := client.Post(srv.URL+"/logout", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("POST /logout: got %d, want 303", resp.StatusCode)
	}

	for _, c := range resp.Cookies() {
		if c.Name == "auth_token" && c.MaxAge >= 0 {
			t.Errorf("expected auth_token cookie to be cleared, got MaxAge=%d", c.MaxAge)
		}
	}
}

// singleCookieJar is a minimal http.CookieJar that always sends one cookie.
type singleCookieJar struct {
	name, value, host string
}

func (j *singleCookieJar) SetCookies(*url.URL, []*http.Cookie) {}
func (j *singleCookieJar) Cookies(u *url.URL) []*http.Cookie {
	if strings.HasPrefix(u.String(), j.host) {
		return []*http.Cookie{{Name: j.name, Value: j.value}}
	}
	return nil
}
