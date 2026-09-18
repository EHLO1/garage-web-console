package router

import (
	"encoding/json"
	"io"
	"khairul169/garage-webui/utils"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// Covers the bcrypt and SCS upgrades through the application's real handlers,
// cookie middleware, and persisted user store rather than mocking the libraries.
func TestPasswordAndSessionLifecycle(t *testing.T) {
	t.Setenv("USERS_PATH", filepath.Join(t.TempDir(), "users.json"))
	t.Setenv("AUTH_USER_PASS", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	previousUsers, previousSession := utils.Users, utils.Session
	t.Cleanup(func() { utils.Users, utils.Session = previousUsers, previousSession })
	utils.InitUserStore()
	sessions := utils.InitSessionManager()
	server := httptest.NewServer(sessions.LoadAndSave(HandleApiRouter()))
	defer server.Close()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}
	request := func(method, path, body string, status int) []byte {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != status {
			t.Fatalf("%s %s: status=%d, expected=%d: %s", method, path, res.StatusCode, status, data)
		}
		if strings.Contains(string(data), "passwordHash") {
			t.Fatal("response leaked password hash")
		}
		return data
	}
	assertStatus := func(authenticated, needsSetup bool) {
		t.Helper()
		var status struct {
			Authenticated bool
			NeedsSetup    bool
		}
		if err := json.Unmarshal(request("GET", "/auth/status", "", 200), &status); err != nil {
			t.Fatal(err)
		}
		if status.Authenticated != authenticated || status.NeedsSetup != needsSetup {
			t.Fatalf("unexpected auth status: %+v", status)
		}
	}
	assertStatus(false, true)
	request("POST", "/auth/register", `{"username":"owner","password":"first-password"}`, 200)
	assertStatus(true, false)
	request("POST", "/auth/register", `{"username":"another","password":"password"}`, 403)
	request("POST", "/auth/change-password", `{"currentPassword":"wrong","newPassword":"new-password"}`, 401)
	request("POST", "/auth/change-password", `{"currentPassword":"first-password","newPassword":"new-password"}`, 200)
	request("POST", "/auth/logout", "", 200)
	assertStatus(false, false)
	request("GET", "/users", "", 401)
	request("POST", "/auth/login", `{"username":"owner","password":"first-password"}`, 401)
	// Reload from disk to check that the updated bcrypt hash persists.
	utils.InitUserStore()
	request("POST", "/auth/login", `{"username":"owner","password":"new-password"}`, 200)
	assertStatus(true, false)
	request("GET", "/users", "", 200)
}
