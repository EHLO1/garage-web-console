package router

import (
	"encoding/json"
	"fmt"
	"khairul169/garage-webui/schema"
	"khairul169/garage-webui/utils"
	"net/http"
	"net/url"
)

type Buckets struct{}

func (b *Buckets) GetAll(w http.ResponseWriter, r *http.Request) {
	body, err := utils.Garage.Fetch("/v2/ListBuckets", &utils.FetchOptions{})
	if err != nil {
		utils.ResponseError(w, err)
		return
	}

	var buckets []schema.GetBucketsRes
	if err := json.Unmarshal(body, &buckets); err != nil {
		utils.ResponseError(w, err)
		return
	}

	ch := make(chan schema.Bucket, len(buckets))

	for _, bucket := range buckets {
		go func() {
			body, err := utils.Garage.Fetch(fmt.Sprintf("/v2/GetBucketInfo?id=%s", bucket.ID), &utils.FetchOptions{})

			if err != nil {
				ch <- schema.Bucket{ID: bucket.ID, GlobalAliases: bucket.GlobalAliases}
				return
			}

			var data schema.Bucket
			if err := json.Unmarshal(body, &data); err != nil {
				ch <- schema.Bucket{ID: bucket.ID, GlobalAliases: bucket.GlobalAliases}
				return
			}

			data.LocalAliases = bucket.LocalAliases
			ch <- data
		}()
	}

	res := make([]schema.Bucket, 0, len(buckets))
	for i := 0; i < len(buckets); i++ {
		res = append(res, <-ch)
	}

	// Non-admins only see the buckets assigned to them.
	if user, ok := utils.GetCurrentUser(r); ok && user.Role != schema.RoleAdmin {
		filtered := make([]schema.Bucket, 0, len(res))
		for _, bucket := range res {
			if user.HasBucket(bucket.ID) {
				filtered = append(filtered, bucket)
			}
		}
		res = filtered
	}

	user, _ := utils.GetCurrentUser(r)
	for i := range res {
		prepareBucket(&res[i], user.Role)
	}
	utils.ResponseSuccess(w, res)
}

func prepareBucket(bucket *schema.Bucket, role schema.Role) {
	for _, key := range bucket.Keys {
		if key.Permissions.Read && key.Permissions.Write {
			bucket.BrowseAvailable = true
		}
	}
	if role != schema.RoleAdmin {
		bucket.Keys = nil
		bucket.LocalAliases = nil
	}
}

func (b *Buckets) GetOne(w http.ResponseWriter, r *http.Request) {
	query := url.Values{}
	if id := r.URL.Query().Get("id"); id != "" {
		query.Set("id", id)
	} else {
		query.Set("globalAlias", r.URL.Query().Get("globalAlias"))
	}
	body, err := utils.Garage.Fetch("/v2/GetBucketInfo?"+query.Encode(), &utils.FetchOptions{})
	if err != nil {
		utils.ResponseError(w, err)
		return
	}
	var bucket schema.Bucket
	if err := json.Unmarshal(body, &bucket); err != nil {
		utils.ResponseError(w, err)
		return
	}
	user, _ := utils.GetCurrentUser(r)
	if user.Role != schema.RoleAdmin && !user.HasBucket(bucket.ID) {
		utils.ResponseErrorStatus(w, fmt.Errorf("bucket access denied"), http.StatusForbidden)
		return
	}
	prepareBucket(&bucket, user.Role)
	utils.ResponseSuccess(w, bucket)
}
