package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"khairul169/garage-webui/schema"
	"khairul169/garage-webui/utils"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(utils.Garage.GetAdminEndpoint())
	if err != nil {
		utils.ResponseError(w, err)
		return
	}

	proxy := &httputil.ReverseProxy{
		ModifyResponse: func(response *http.Response) error {
			user, _ := utils.GetCurrentUser(r)
			if user.Role == schema.RoleAdmin || response.StatusCode >= 400 {
				return nil
			}
			// Bucket mutations can return bucket metadata. Keep key details admin-only.
			data, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				return err
			}
			var object map[string]json.RawMessage
			if json.Unmarshal(data, &object) == nil && object != nil {
				delete(object, "keys")
				delete(object, "localAliases")
				delete(object, "secretAccessKey")
				data, err = json.Marshal(object)
				if err != nil {
					return err
				}
			}
			response.Body = io.NopCloser(bytes.NewReader(data))
			response.ContentLength = int64(len(data))
			response.Header.Set("Content-Length", fmt.Sprint(len(data)))
			return nil
		},
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.URL.Path = strings.TrimPrefix(r.In.URL.Path, "/api")
			r.Out.Header.Set("Authorization", fmt.Sprintf("Bearer %s", utils.Garage.GetAdminKey()))
			if user, ok := utils.GetCurrentUser(r.In); ok && user.Role != schema.RoleAdmin {
				// Let the transport negotiate/decompress responses before redaction.
				r.Out.Header.Del("Accept-Encoding")
			}
		},
	}

	proxy.ServeHTTP(w, r)
}
