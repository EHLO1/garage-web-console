package utils

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCConfig describes one trusted provider. Secrets never go to the browser.
type OIDCConfig struct {
	Issuer, ClientID, ClientSecret, RedirectURL string
	Scopes                                      []string
	AllowedDomains                              []string
	RequireVerifiedEmail                        bool
}

func LoadOIDCConfig() (OIDCConfig, error) {
	c := OIDCConfig{
		Issuer:               strings.TrimSpace(os.Getenv("OIDC_ISSUER")),
		ClientID:             strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")),
		ClientSecret:         os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:          strings.TrimSpace(os.Getenv("OIDC_REDIRECT_URL")),
		RequireVerifiedEmail: true,
	}
	if c.Issuer == "" || c.ClientID == "" || c.RedirectURL == "" {
		return c, errors.New("OIDC_ISSUER, OIDC_CLIENT_ID and OIDC_REDIRECT_URL are required")
	}
	for _, raw := range []string{c.Issuer, c.RedirectURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))) {
			return c, errors.New("OIDC URLs must use HTTPS (HTTP is allowed only on loopback for development)")
		}
	}
	switch os.Getenv("OIDC_REQUIRE_VERIFIED_EMAIL") {
	case "", "true":
	case "false":
		c.RequireVerifiedEmail = false
	default:
		return c, errors.New("OIDC_REQUIRE_VERIFIED_EMAIL must be true or false")
	}
	c.Scopes = []string{oidc.ScopeOpenID}
	for _, scope := range strings.Fields(strings.ReplaceAll(GetEnv("OIDC_SCOPES", "email profile"), ",", " ")) {
		if scope != oidc.ScopeOpenID {
			c.Scopes = append(c.Scopes, scope)
		}
	}
	for _, domain := range strings.Split(os.Getenv("OIDC_ALLOWED_DOMAINS"), ",") {
		if domain = strings.ToLower(strings.TrimSpace(domain)); domain != "" {
			c.AllowedDomains = append(c.AllowedDomains, domain)
		}
	}
	return c, nil
}

func IsOIDCEnabled() bool { _, err := LoadOIDCConfig(); return err == nil }

func OIDCButtonText() string {
	if text := strings.TrimSpace(os.Getenv("OIDC_BUTTON_TEXT")); text != "" {
		return text
	}
	return "Continue with OpenID Connect"
}

// Render icons as images, never as administrator-supplied HTML or SVG markup.
func OIDCButtonIconURL() string {
	raw := strings.TrimSpace(os.Getenv("OIDC_BUTTON_ICON_URL"))
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || strings.Contains(raw, "\\") {
		return ""
	}
	if u.Scheme == "https" && u.Host != "" {
		return raw
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") && u.Host == "" {
		return raw
	}
	return ""
}

var oidcCache struct {
	sync.Mutex
	issuer   string
	provider *oidc.Provider
}

var oidcHTTPClient = &http.Client{Timeout: 15 * time.Second}

// Failed discovery is not cached, allowing recovery after a provider outage.
func OIDCProvider(ctx context.Context, c OIDCConfig) (*oidc.Provider, *oidc.IDTokenVerifier, error) {
	oidcCache.Lock()
	defer oidcCache.Unlock()
	if oidcCache.provider == nil || oidcCache.issuer != c.Issuer {
		provider, err := oidc.NewProvider(oidc.ClientContext(ctx, oidcHTTPClient), c.Issuer)
		if err != nil {
			return nil, nil, err
		}
		oidcCache.provider, oidcCache.issuer = provider, c.Issuer
	}
	return oidcCache.provider, oidcCache.provider.Verifier(&oidc.Config{ClientID: c.ClientID}), nil
}
