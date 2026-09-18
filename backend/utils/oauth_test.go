package utils

import "testing"

func TestOIDCConfiguration(t *testing.T) {
	for k, v := range map[string]string{"OIDC_ISSUER": "https://id.example", "OIDC_CLIENT_ID": "client", "OIDC_CLIENT_SECRET": "", "OIDC_REDIRECT_URL": "https://console.example/api/v1/auth/oidc/callback", "OIDC_SCOPES": "", "OIDC_REQUIRE_VERIFIED_EMAIL": "", "OIDC_ALLOWED_DOMAINS": ""} {
		t.Setenv(k, v)
	}
	c, err := LoadOIDCConfig()
	if err != nil || !c.RequireVerifiedEmail || len(c.Scopes) != 3 || !IsOIDCEnabled() {
		t.Fatalf("invalid defaults: %+v %v", c, err)
	}
	for _, tc := range []struct{ key, value string }{
		{"OIDC_ISSUER", ""}, {"OIDC_CLIENT_ID", ""}, {"OIDC_REDIRECT_URL", ""},
		{"OIDC_ISSUER", "http://id.example"}, {"OIDC_REDIRECT_URL", "javascript:alert(1)"},
		{"OIDC_ISSUER", "https://user:pass@id.example"}, {"OIDC_REQUIRE_VERIFIED_EMAIL", "typo"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if IsOIDCEnabled() {
				t.Fatal("invalid config enabled")
			}
		})
	}
	t.Setenv("OIDC_SCOPES", "openid,email profile groups")
	t.Setenv("OIDC_ALLOWED_DOMAINS", " EXAMPLE.COM, ,other.example ")
	c, err = LoadOIDCConfig()
	if err != nil || len(c.Scopes) != 4 || len(c.AllowedDomains) != 2 || c.AllowedDomains[0] != "example.com" {
		t.Fatalf("invalid custom config: %+v %v", c, err)
	}
}

func TestOIDCButtonBranding(t *testing.T) {
	t.Setenv("OIDC_BUTTON_TEXT", "")
	if OIDCButtonText() != "Continue with OpenID Connect" {
		t.Fatal("missing default label")
	}
	t.Setenv("OIDC_BUTTON_TEXT", " Sign in with Pocket ID ")
	if OIDCButtonText() != "Sign in with Pocket ID" {
		t.Fatal("custom label missing")
	}
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"https://id.example/logo.svg", true}, {"/console/favicon.ico", true},
		{"//other.example/logo", false}, {"javascript:alert(1)", false},
		{"data:image/svg+xml,<svg/>", false}, {"/\\other.example/logo", false},
		{"https://user:password@id.example/logo", false}, {"", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("OIDC_BUTTON_ICON_URL", tc.value)
			if (OIDCButtonIconURL() != "") != tc.valid {
				t.Fatal("unexpected icon URL validation")
			}
		})
	}
}
