package router

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"khairul169/garage-webui/utils"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type OIDC struct{}

func oidcOAuthConfig(provider *oidc.Provider, c utils.OIDCConfig) *oauth2.Config {
	endpoint := provider.Endpoint()
	if c.ClientSecret == "" {
		endpoint.AuthStyle = oauth2.AuthStyleInParams
	}
	return &oauth2.Config{ClientID: c.ClientID, ClientSecret: c.ClientSecret, Endpoint: endpoint, RedirectURL: c.RedirectURL, Scopes: c.Scopes}
}

func loginRedirect(w http.ResponseWriter, r *http.Request, message string) {
	target := os.Getenv("BASE_PATH") + "/auth/login"
	if message != "" {
		target += "?error=" + url.QueryEscape(message)
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (o *OIDC) Login(w http.ResponseWriter, r *http.Request) {
	c, err := utils.LoadOIDCConfig()
	if err != nil {
		utils.ResponseErrorStatus(w, errors.New("OIDC sign-in is not configured"), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	provider, _, err := utils.OIDCProvider(ctx, c)
	if err != nil {
		utils.ResponseErrorStatus(w, errors.New("OIDC sign-in unavailable"), http.StatusBadGateway)
		return
	}
	state, nonce, verifier := oauth2.GenerateVerifier(), oauth2.GenerateVerifier(), oauth2.GenerateVerifier()
	utils.Session.Set(r, "oidcState", state)
	utils.Session.Set(r, "oidcNonce", nonce)
	utils.Session.Set(r, "oidcVerifier", verifier)
	utils.Session.Set(r, "oidcExpires", time.Now().Add(10*time.Minute).Unix())
	utils.Session.Set(r, "oidcIssuer", c.Issuer)
	utils.Session.Set(r, "oidcClientID", c.ClientID)
	utils.Session.Set(r, "oidcRedirectURL", c.RedirectURL)
	conf := oidcOAuthConfig(provider, c)
	http.Redirect(w, r, conf.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}

func (o *OIDC) Callback(w http.ResponseWriter, r *http.Request) {
	c, err := utils.LoadOIDCConfig()
	if err != nil {
		loginRedirect(w, r, "OIDC sign-in is not configured")
		return
	}
	state, _ := utils.Session.Get(r, "oidcState").(string)
	nonce, _ := utils.Session.Get(r, "oidcNonce").(string)
	pkce, _ := utils.Session.Get(r, "oidcVerifier").(string)
	expires, _ := utils.Session.Get(r, "oidcExpires").(int64)
	issuer, _ := utils.Session.Get(r, "oidcIssuer").(string)
	clientID, _ := utils.Session.Get(r, "oidcClientID").(string)
	redirectURL, _ := utils.Session.Get(r, "oidcRedirectURL").(string)
	for _, key := range []string{"oidcState", "oidcNonce", "oidcVerifier", "oidcExpires", "oidcIssuer", "oidcClientID", "oidcRedirectURL"} {
		utils.Session.Remove(r, key)
	}
	if state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(r.URL.Query().Get("state"))) != 1 || nonce == "" || pkce == "" || time.Now().Unix() >= expires || issuer != c.Issuer || clientID != c.ClientID || redirectURL != c.RedirectURL {
		loginRedirect(w, r, "invalid or expired sign-in state, please try again")
		return
	}
	if r.URL.Query().Get("error") != "" {
		loginRedirect(w, r, "OIDC sign-in was cancelled")
		return
	}
	if r.URL.Query().Get("code") == "" {
		loginRedirect(w, r, "missing sign-in code")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	provider, verifier, err := utils.OIDCProvider(ctx, c)
	if err != nil {
		loginRedirect(w, r, "OIDC sign-in unavailable")
		return
	}
	token, err := oidcOAuthConfig(provider, c).Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(pkce))
	if err != nil {
		loginRedirect(w, r, "could not complete OIDC sign-in")
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		loginRedirect(w, r, "provider did not return an identity token")
		return
	}
	identity, err := verifier.Verify(ctx, raw)
	if err != nil || identity.Subject == "" {
		loginRedirect(w, r, "could not verify OIDC identity")
		return
	}
	if identity.Nonce != nonce {
		loginRedirect(w, r, "invalid sign-in nonce, please try again")
		return
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
	}
	if err := identity.Claims(&claims); err != nil {
		loginRedirect(w, r, "could not read account details")
		return
	}
	// Some providers expose email only through UserInfo. Bind it to the verified ID token.
	if claims.Email == "" || (c.RequireVerifiedEmail && claims.EmailVerified == nil) {
		info, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err != nil || info.Subject != identity.Subject {
			loginRedirect(w, r, "could not verify provider account details")
			return
		}
		claims.Email = info.Email
		claims.EmailVerified = &info.EmailVerified
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		loginRedirect(w, r, "provider did not return a valid email address")
		return
	}
	if c.RequireVerifiedEmail && (claims.EmailVerified == nil || !*claims.EmailVerified) {
		loginRedirect(w, r, "your email is not verified")
		return
	}
	if !isAllowedOIDCDomain(email, c.AllowedDomains) {
		utils.Audit(r, "WARN", "OIDC sign-in denied (domain not permitted)", map[string]interface{}{"event": "oidc_denied", "email": email})
		loginRedirect(w, r, "your account domain is not permitted")
		return
	}
	user, ok := utils.Users.GetByEmail(email)
	if !ok {
		utils.Audit(r, "WARN", "OIDC sign-in denied (no registered account)", map[string]interface{}{"event": "oidc_denied", "email": email})
		loginRedirect(w, r, "no account is registered for this email. Please ask an administrator to add you.")
		return
	}
	if err := utils.Session.RenewToken(r); err != nil {
		loginRedirect(w, r, "could not create sign-in session")
		return
	}
	utils.Session.Set(r, "userId", user.ID)
	utils.Audit(r, "INFO", fmt.Sprintf("User %s signed in with OIDC", user.Username), map[string]interface{}{"event": "oidc_login", "email": email, "issuer": identity.Issuer})
	http.Redirect(w, r, os.Getenv("BASE_PATH")+"/", http.StatusFound)
}

func isAllowedOIDCDomain(email string, domains []string) bool {
	if len(domains) == 0 {
		return true
	}
	_, domain, ok := strings.Cut(email, "@")
	if !ok {
		return false
	}
	for _, allowed := range domains {
		if domain == allowed {
			return true
		}
	}
	return false
}
