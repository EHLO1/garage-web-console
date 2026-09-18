package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"khairul169/garage-webui/schema"
	"khairul169/garage-webui/utils"
	"net/http"
	"net/url"
	"strings"
)

var errForbidden = errors.New("you do not have permission to access this resource")

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := utils.GetCurrentUser(r)
		if !ok {
			utils.ResponseErrorStatus(w, errors.New("unauthorized"), http.StatusUnauthorized)
			return
		}
		if err := authorize(user, r); err != nil {
			utils.ResponseErrorStatus(w, err, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authorize(user schema.User, r *http.Request) error {
	if user.Role == schema.RoleAdmin {
		return nil
	}
	if user.Role != schema.RoleUser && user.Role != schema.RoleViewer {
		return errForbidden
	}
	path, method := r.URL.Path, r.Method
	if method == http.MethodGet && path == "/buckets" {
		return nil
	}
	if strings.HasPrefix(path, "/browse/") {
		if method != http.MethodGet && (user.Role != schema.RoleUser || (method != http.MethodPut && method != http.MethodPost && method != http.MethodDelete)) {
			return errForbidden
		}
		if canAccessBucket(user, bucketFromBrowsePath(r.URL.EscapedPath())) {
			return nil
		}
		return errForbidden
	}
	if method == http.MethodGet && path == "/v2/GetBucketInfo" {
		// Exactly one selector avoids ambiguous upstream query interpretation.
		q := r.URL.Query()
		if len(q) == 1 && len(q["id"]) == 1 && user.HasBucket(q.Get("id")) {
			return nil
		}
		if len(q) == 1 && len(q["globalAlias"]) == 1 && canAccessBucket(user, q.Get("globalAlias")) {
			return nil
		}
		return errForbidden
	}
	if user.Role != schema.RoleUser || method != http.MethodPost {
		return errForbidden
	}
	switch path {
	case "/v2/UpdateBucket", "/v2/DeleteBucket":
		q := r.URL.Query()
		if len(q) != 1 || len(q["id"]) != 1 || !user.HasBucket(q.Get("id")) {
			return errForbidden
		}
		if path == "/v2/DeleteBucket" {
			return nil
		}
		// Only bucket-local settings are allowed. Reject unknown or duplicate fields.
		body, err := readObject(r)
		if err != nil {
			return errForbidden
		}
		for key := range body {
			if key != "quotas" && key != "websiteAccess" {
				return errForbidden
			}
		}
		return nil
	case "/v2/AddBucketAlias", "/v2/RemoveBucketAlias":
		if r.URL.RawQuery != "" {
			return errForbidden
		}
		body, err := readObject(r)
		if err != nil || len(body) != 2 {
			return errForbidden
		}
		var bucketID, alias string
		if json.Unmarshal(body["bucketId"], &bucketID) != nil || json.Unmarshal(body["globalAlias"], &alias) != nil || alias == "" || !user.HasBucket(bucketID) {
			return errForbidden
		}
		return nil
	}
	// Key APIs, grants, cluster operations, config, users, logs, and new endpoints
	// are denied unless explicitly allowed above.
	return errForbidden
}

// Preserve the validated body for the proxy; reject duplicate top-level keys.
func readObject(r *http.Request) (map[string]json.RawMessage, error) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 65537))
	if err != nil || len(data) > 65536 {
		return nil, errForbidden
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	dec := json.NewDecoder(bytes.NewReader(data))
	token, err := dec.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errForbidden
	}
	result := map[string]json.RawMessage{}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, errForbidden
		}
		if _, exists := result[name]; exists {
			return nil, errForbidden
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errForbidden
	}
	return result, nil
}

func bucketFromBrowsePath(path string) string {
	rest := strings.TrimPrefix(path, "/browse/")
	name, _, _ := strings.Cut(rest, "/")
	decoded, err := url.PathUnescape(name)
	if err != nil {
		return ""
	}
	return decoded
}

func canAccessBucket(user schema.User, aliasOrID string) bool {
	if aliasOrID == "" {
		return false
	}
	// Resolve aliases fresh: reassignment must not retain access to the old bucket.
	body, err := utils.Garage.Fetch("/v2/GetBucketInfo?globalAlias="+url.QueryEscape(aliasOrID), &utils.FetchOptions{})
	if err != nil {
		return false
	}
	var info struct {
		ID string `json:"id"`
	}
	return json.Unmarshal(body, &info) == nil && user.HasBucket(info.ID)
}
