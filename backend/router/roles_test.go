package router

import (
	"encoding/json"
	"io"
	"khairul169/garage-webui/schema"
	"khairul169/garage-webui/utils"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestScopedRolesThroughRouter(t *testing.T) {
	var upstreamCalls atomic.Int64
	bucket := func(id string) map[string]interface{} {
		return map[string]interface{}{"id": id, "globalAliases": []string{"photos"}, "keys": []interface{}{map[string]interface{}{"accessKeyId": "shared-key", "secretAccessKey": "private-secret", "permissions": map[string]bool{"read": true, "write": true}}}}
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/ListBuckets":
			json.NewEncoder(w).Encode([]interface{}{bucket("allowed"), bucket("other")})
		case "/v2/GetBucketInfo":
			id := r.URL.Query().Get("id")
			if id == "" {
				id = "allowed"
			}
			json.NewEncoder(w).Encode(bucket(id))
		case "/v2/GetKeyInfo":
			json.NewEncoder(w).Encode(map[string]string{"accessKeyId": "shared-key", "secretAccessKey": "private-secret"})
		case "/v2/UpdateBucket", "/v2/AddBucketAlias":
			json.NewEncoder(w).Encode(bucket("allowed"))
		case "/photos/file.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Last-Modified", "Wed, 01 Jan 2025 00:00:00 GMT")
			io.WriteString(w, "downloaded object")
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	t.Setenv("API_BASE_URL", upstream.URL)
	t.Setenv("S3_ENDPOINT_URL", upstream.URL)
	t.Setenv("S3_REGION", "garage")
	t.Setenv("USERS_PATH", filepath.Join(t.TempDir(), "users.json"))
	oldUsers, oldSession, oldCache := utils.Users, utils.Session, utils.Cache
	t.Cleanup(func() { utils.Users, utils.Session, utils.Cache = oldUsers, oldSession, oldCache })
	utils.InitUserStore()
	utils.InitCacheManager()
	for _, role := range []schema.Role{schema.RoleAdmin, schema.RoleUser, schema.RoleViewer} {
		if _, err := utils.Users.Create(schema.User{ID: string(role), Username: string(role), Role: role, Buckets: []string{"allowed"}}); err != nil {
			t.Fatal(err)
		}
	}
	sessions := utils.InitSessionManager()
	router := HandleApiRouter()
	app := httptest.NewServer(sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if role := r.Header.Get("X-Test-Account"); role != "" {
			utils.Session.Set(r, "userId", role)
		}
		router.ServeHTTP(w, r)
	})))
	defer app.Close()
	request := func(role, method, path, body string, status int) string {
		t.Helper()
		req, err := http.NewRequest(method, app.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Test-Account", role)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Encoding", "gzip")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode != status {
			t.Fatalf("%s %s %s: %d %s", role, method, path, res.StatusCode, data)
		}
		return string(data)
	}
	request("", "GET", "/buckets", "", 401)
	for _, role := range []string{"user", "viewer"} {
		for _, path := range []string{"/buckets", "/v2/GetBucketInfo?id=allowed"} {
			data := request(role, "GET", path, "", 200)
			if strings.Contains(data, "shared-key") || strings.Contains(data, "private-secret") || strings.Contains(data, `"id":"other"`) {
				t.Fatalf("restricted metadata leaked: %s", data)
			}
			if !strings.Contains(data, `"browseAvailable":true`) {
				t.Fatal("browser capability missing")
			}
		}
		if data := request(role, "GET", "/browse/photos/file.txt?dl=1", "", 200); data != "downloaded object" {
			t.Fatalf("download failed: %s", data)
		}
		before := upstreamCalls.Load()
		request(role, "GET", "/v2/GetKeyInfo?id=shared-key&showSecretKey=true", "", 403)
		request(role, "GET", "/v2/GetBucketInfo?id=other", "", 403)
		request(role, "POST", "/v2/AllowBucketKey", `{"bucketId":"allowed"}`, 403)
		request(role, "GET", "/config", "", 403)
		request(role, "PATCH", "/users/admin", `{"role":"viewer"}`, 403)
		if upstreamCalls.Load() != before {
			t.Fatal("forbidden request reached upstream")
		}
	}
	before := upstreamCalls.Load()
	request("viewer", "PUT", "/browse/photos/file.txt", "replacement", 403)
	request("viewer", "DELETE", "/browse/photos/file.txt", "", 403)
	request("viewer", "POST", "/v2/UpdateBucket?id=allowed", `{"quotas":{}}`, 403)
	if upstreamCalls.Load() != before {
		t.Fatal("viewer mutation reached upstream")
	}
	data := request("user", "POST", "/v2/UpdateBucket?id=allowed", `{"quotas":{"maxObjects":100}}`, 200)
	if strings.Contains(data, "shared-key") || strings.Contains(data, "private-secret") {
		t.Fatal("mutation response leaked key")
	}
	if !strings.Contains(request("admin", "GET", "/v2/GetKeyInfo?id=shared-key&showSecretKey=true", "", 200), "private-secret") {
		t.Fatal("admin key access denied")
	}
	request("admin", "PATCH", "/users/admin", `{"role":"viewer"}`, 400)
}
