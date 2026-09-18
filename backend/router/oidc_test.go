package router

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"khairul169/garage-webui/schema"
	"khairul169/garage-webui/utils"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOIDCFlow(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var issuer, nonce, challenge string
	var tokenClaims map[string]interface{}
	var userInfo map[string]interface{}
	var tokenRequests int
	var discoveryUnavailable bool
	var publicClient bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			if discoveryUnavailable {
				http.Error(w, "unavailable", 503)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "userinfo_endpoint": issuer + "/userinfo", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{map[string]interface{}{"kty": "RSA", "kid": "test", "use": "sig", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
		case "/token":
			tokenRequests++
			r.ParseForm()
			if publicClient && (r.Header.Get("Authorization") != "" || r.Form.Get("client_id") != "console-client" || r.Form.Get("client_secret") != "") {
				t.Error("incorrect public client authentication")
			}
			sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge || r.Form.Get("code_verifier") == "" {
				t.Error("missing or incorrect PKCE verifier")
				http.Error(w, "PKCE", 400)
				return
			}
			if r.Form.Get("redirect_uri") != "https://console.example/console/api/v1/auth/oidc/callback" {
				t.Error("callback changed during exchange")
			}
			claims := map[string]interface{}{"iss": issuer, "sub": "subject-1", "aud": "console-client", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": nonce, "email": "OWNER@example.com", "email_verified": true}
			for k, v := range tokenClaims {
				if v == nil {
					delete(claims, k)
				} else {
					claims[k] = v
				}
			}
			payload, _ := json.Marshal(claims)
			header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"test"}`))
			data := header + "." + base64.RawURLEncoding.EncodeToString(payload)
			digest := sha256.Sum256([]byte(data))
			sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
			if err != nil {
				t.Error(err)
				return
			}
			if tokenClaims["badSignature"] == true {
				sig[0] ^= 1
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "access-token", "token_type": "Bearer", "id_token": data + "." + base64.RawURLEncoding.EncodeToString(sig)})
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer access-token" {
				t.Error("missing userinfo bearer token")
			}
			json.NewEncoder(w).Encode(userInfo)
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	for k, v := range map[string]string{"OIDC_ISSUER": issuer, "OIDC_CLIENT_ID": "console-client", "OIDC_CLIENT_SECRET": "client-secret", "OIDC_REDIRECT_URL": "https://console.example/console/api/v1/auth/oidc/callback", "OIDC_REQUIRE_VERIFIED_EMAIL": "true", "OIDC_ALLOWED_DOMAINS": "", "OIDC_SCOPES": "email profile", "BASE_PATH": "/console", "USERS_PATH": filepath.Join(t.TempDir(), "users.json"), "OIDC_BUTTON_TEXT": "Sign in with Pocket ID", "OIDC_BUTTON_ICON_URL": "https://id.example/icon.svg"} {
		t.Setenv(k, v)
	}
	previousUsers, previousSession := utils.Users, utils.Session
	t.Cleanup(func() { utils.Users, utils.Session = previousUsers, previousSession })
	utils.InitUserStore()
	_, err = utils.Users.Create(schema.User{Username: "owner", Email: "owner@example.com", Role: schema.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	sessions := utils.InitSessionManager()
	mux := HandleApiRouter()
	mux.HandleFunc("/test/expire", func(w http.ResponseWriter, r *http.Request) {
		utils.Session.Set(r, "oidcExpires", time.Now().Add(-time.Minute).Unix())
	})
	app := httptest.NewServer(sessions.LoadAndSave(mux))
	defer app.Close()
	var client *http.Client
	resetClient := func() {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	resetClient()
	get := func(path string) *http.Response {
		t.Helper()
		res, err := client.Get(app.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	start := func() string {
		t.Helper()
		res := get("/v1/auth/oidc/login")
		defer res.Body.Close()
		if res.StatusCode != 302 {
			body, _ := io.ReadAll(res.Body)
			t.Fatalf("login: %d %s", res.StatusCode, body)
		}
		target, err := url.Parse(res.Header.Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		q := target.Query()
		nonce = q.Get("nonce")
		challenge = q.Get("code_challenge")
		if q.Get("state") == "" || nonce == "" || challenge == "" || q.Get("code_challenge_method") != "S256" || q.Get("scope") != "openid email profile" {
			t.Fatalf("invalid authorization request: %v", q)
		}
		if q.Get("redirect_uri") != "https://console.example/console/api/v1/auth/oidc/callback" {
			t.Fatal("incorrect redirect URI")
		}
		return q.Get("state")
	}
	callback := func(state string) string {
		t.Helper()
		res := get("/v1/auth/oidc/callback?code=code&state=" + url.QueryEscape(state))
		defer res.Body.Close()
		return res.Header.Get("Location")
	}
	authenticated := func() bool {
		t.Helper()
		res := get("/auth/status")
		defer res.Body.Close()
		var data struct {
			Authenticated     bool
			OIDCButtonText    string
			OIDCButtonIconURL string
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "client-secret") || strings.Contains(string(body), "access-token") {
			t.Fatal("credentials leaked through auth status")
		}
		if err := json.Unmarshal(body, &data); err != nil {
			t.Fatal(err)
		}
		if data.OIDCButtonText != "Sign in with Pocket ID" || data.OIDCButtonIconURL != "https://id.example/icon.svg" {
			t.Fatal("missing public branding")
		}
		return data.Authenticated
	}
	// Discovery failures must be retryable, rather than cached forever.
	discoveryUnavailable = true
	res := get("/v1/auth/oidc/login")
	res.Body.Close()
	if res.StatusCode != 502 {
		t.Fatal("expected unavailable discovery")
	}
	discoveryUnavailable = false
	for _, tc := range []struct {
		name           string
		claims         map[string]interface{}
		info           map[string]interface{}
		allow, require string
		success        bool
	}{
		{name: "verified email", success: true},
		{name: "UserInfo email", claims: map[string]interface{}{"email": nil, "email_verified": nil}, info: map[string]interface{}{"sub": "subject-1", "email": "owner@example.com", "email_verified": true}, success: true},
		{name: "UserInfo subject mismatch", claims: map[string]interface{}{"email": nil}, info: map[string]interface{}{"sub": "other", "email": "owner@example.com", "email_verified": true}},
		{name: "unverified email", claims: map[string]interface{}{"email_verified": false}},
		{name: "missing verification", claims: map[string]interface{}{"email_verified": nil}, info: map[string]interface{}{"sub": "subject-1", "email": "owner@example.com"}},
		{name: "explicit trusted-provider opt-out", claims: map[string]interface{}{"email_verified": nil}, require: "false", success: true},
		{name: "unknown account", claims: map[string]interface{}{"email": "unknown@example.com"}},
		{name: "missing email", claims: map[string]interface{}{"email": nil}, info: map[string]interface{}{"sub": "subject-1", "email_verified": true}},
		{name: "wrong nonce", claims: map[string]interface{}{"nonce": "wrong"}},
		{name: "wrong audience", claims: map[string]interface{}{"aud": "different-client"}},
		{name: "wrong issuer", claims: map[string]interface{}{"iss": "https://other.example"}},
		{name: "expired token", claims: map[string]interface{}{"exp": time.Now().Add(-time.Hour).Unix()}},
		{name: "invalid signature", claims: map[string]interface{}{"badSignature": true}},
		{name: "domain rejected", allow: "elsewhere.example"},
		{name: "domain allowed", allow: "EXAMPLE.COM", success: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetClient()
			tokenClaims = tc.claims
			userInfo = tc.info
			t.Setenv("OIDC_ALLOWED_DOMAINS", tc.allow)
			if tc.require != "" {
				t.Setenv("OIDC_REQUIRE_VERIFIED_EMAIL", tc.require)
			}
			state := start()
			before := client.Jar.Cookies(mustURL(t, app.URL))
			location := callback(state)
			if (location == "/console/") != tc.success || authenticated() != tc.success {
				t.Fatalf("unexpected result %s", location)
			}
			if tc.success {
				after := client.Jar.Cookies(mustURL(t, app.URL))
				if len(before) == 0 || len(after) == 0 || before[0].Value == after[0].Value {
					t.Fatal("session token was not rotated")
				}
			}
			requests := tokenRequests
			if !strings.Contains(callback(state), "error=") || tokenRequests != requests {
				t.Fatal("callback replay was accepted")
			}
		})
	}
	t.Run("state mismatch", func(t *testing.T) {
		resetClient()
		start()
		requests := tokenRequests
		if !strings.Contains(callback("wrong"), "error=") || authenticated() || tokenRequests != requests {
			t.Fatal("invalid state accepted")
		}
	})
	t.Run("public client", func(t *testing.T) {
		resetClient()
		tokenClaims = nil
		publicClient = true
		defer func() { publicClient = false }()
		t.Setenv("OIDC_CLIENT_SECRET", "")
		if callback(start()) != "/console/" || !authenticated() {
			t.Fatal("public client failed")
		}
	})
	t.Run("expired state", func(t *testing.T) {
		resetClient()
		state := start()
		res := get("/test/expire")
		res.Body.Close()
		requests := tokenRequests
		if !strings.Contains(callback(state), "error=") || authenticated() || requests != tokenRequests {
			t.Fatal("expired state accepted")
		}
	})
	t.Run("provider cancellation", func(t *testing.T) {
		resetClient()
		state := start()
		requests := tokenRequests
		res := get("/v1/auth/oidc/callback?error=access_denied&state=" + state)
		defer res.Body.Close()
		if !strings.Contains(res.Header.Get("Location"), "cancelled") || authenticated() || tokenRequests != requests {
			t.Fatal("cancellation failed")
		}
	})
	t.Run("disabled callback", func(t *testing.T) {
		resetClient()
		state := start()
		t.Setenv("OIDC_CLIENT_ID", "")
		requests := tokenRequests
		if !strings.Contains(callback(state), "error=") || tokenRequests != requests {
			t.Fatal("disabled callback accepted")
		}
	})
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
