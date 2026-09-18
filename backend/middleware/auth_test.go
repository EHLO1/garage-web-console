package middleware

import (
	"encoding/json"
	"io"
	"khairul169/garage-webui/schema"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRolePermissions(t *testing.T) {
	aliasBucket := "allowed"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/GetBucketInfo" {
			t.Errorf("unexpected upstream request %s", r.URL)
		}
		id := "other"
		if r.URL.Query().Get("globalAlias") == "photos" {
			id = aliasBucket
		}
		json.NewEncoder(w).Encode(map[string]string{"id": id})
	}))
	defer provider.Close()
	t.Setenv("API_BASE_URL", provider.URL)
	for _, tc := range []struct {
		method, path, body string
		user, viewer       bool
	}{
		{"GET", "/buckets", "", true, true},
		{"GET", "/browse/photos", "", true, true},
		{"GET", "/browse/photos/file.txt?dl=1", "", true, true},
		{"GET", "/browse/photos/file.txt?thumb=1", "", true, true},
		{"PUT", "/browse/photos/file.txt", "", true, false},
		{"POST", "/browse/photos", `{"items":["file.txt"],"destination":"folder/"}`, true, false},
		{"DELETE", "/browse/photos/file.txt", "", true, false},
		{"GET", "/browse/private", "", false, false},
		// A global alias resembling an assigned ID must still be resolved.
		{"GET", "/browse/allowed", "", false, false},
		{"GET", "/v2/GetBucketInfo?id=allowed", "", true, true},
		{"GET", "/v2/GetBucketInfo?globalAlias=photos", "", true, true},
		{"GET", "/v2/GetBucketInfo?id=other", "", false, false},
		{"GET", "/v2/GetBucketInfo?id=allowed&id=other", "", false, false},
		{"GET", "/v2/GetBucketInfo?id=allowed&globalAlias=private", "", false, false},
		{"POST", "/v2/UpdateBucket?id=allowed", `{"quotas":{"maxSize":1024}}`, true, false},
		{"POST", "/v2/UpdateBucket?id=allowed", `{"websiteAccess":{"enabled":true}}`, true, false},
		{"POST", "/v2/UpdateBucket?id=other", `{"quotas":{}}`, false, false},
		{"POST", "/v2/UpdateBucket?id=allowed", `{"quotas":{},"quotas":{}}`, false, false},
		{"POST", "/v2/UpdateBucket?id=allowed", `{"keys":["secret"]}`, false, false},
		{"POST", "/v2/UpdateBucket?id=allowed", `{} {}`, false, false},
		{"POST", "/v2/DeleteBucket?id=allowed", "", true, false},
		{"POST", "/v2/DeleteBucket?id=allowed&id=other", "", false, false},
		{"POST", "/v2/AddBucketAlias", `{"bucketId":"allowed","globalAlias":"new"}`, true, false},
		{"POST", "/v2/RemoveBucketAlias", `{"bucketId":"allowed","globalAlias":"photos"}`, true, false},
		{"POST", "/v2/AddBucketAlias", `{"bucketId":"other","globalAlias":"new"}`, false, false},
		{"POST", "/v2/AddBucketAlias", `{"bucketId":"allowed","localAlias":"new","accessKeyId":"key"}`, false, false},
		{"POST", "/v2/AddBucketAlias", `{"bucketId":"allowed","bucketId":"other","globalAlias":"new"}`, false, false},
		{"POST", "/v2/CreateBucket", `{}`, false, false},
		{"GET", "/v2/ListKeys", "", false, false},
		{"GET", "/v2/GetKeyInfo?id=shared&showSecretKey=true", "", false, false},
		{"POST", "/v2/AllowBucketKey", `{"bucketId":"allowed"}`, false, false},
		{"POST", "/v2/CreateKey", `{}`, false, false},
		{"GET", "/config", "", false, false},
		{"GET", "/users", "", false, false},
		{"PATCH", "/users/admin", `{"role":"admin"}`, false, false},
		{"GET", "/logs", "", false, false},
		{"GET", "/v2/GetClusterHealth", "", false, false},
		{"POST", "/v2/FutureEndpoint", `{}`, false, false},
	} {
		for _, role := range []schema.Role{schema.RoleAdmin, schema.RoleUser, schema.RoleViewer, "unknown"} {
			t.Run(string(role)+" "+tc.method+" "+tc.path+" "+tc.body, func(t *testing.T) {
				req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
				err := authorize(schema.User{Role: role, Buckets: []string{"allowed"}}, req)
				want := role == schema.RoleAdmin || (role == schema.RoleUser && tc.user) || (role == schema.RoleViewer && tc.viewer)
				if (err == nil) != want {
					t.Fatalf("authorized=%v want=%v", err == nil, want)
				}
				if want && tc.body != "" {
					body, _ := io.ReadAll(req.Body)
					if string(body) != tc.body {
						t.Fatal("request body was changed")
					}
				}
			})
		}
	}
	user := schema.User{Role: schema.RoleUser, Buckets: []string{"allowed"}}
	if authorize(user, httptest.NewRequest("GET", "/browse/photos", nil)) != nil {
		t.Fatal("initial alias denied")
	}
	aliasBucket = "other"
	if authorize(user, httptest.NewRequest("GET", "/browse/photos", nil)) == nil {
		t.Fatal("stale alias assignment allowed")
	}
	if authorize(schema.User{Role: schema.RoleViewer}, httptest.NewRequest("GET", "/browse/photos", nil)) == nil {
		t.Fatal("empty assignment allowed")
	}
}
