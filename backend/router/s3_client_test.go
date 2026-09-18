package router

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Exercise real SDK serialization, endpoint resolution, and signing against
// an S3-compatible HTTP server. No live Garage instance or credentials needed.
func TestS3ClientGarageCompatibility(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "HTTP", true: "HTTPS"}[secure], func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if !strings.HasPrefix(r.URL.Path, "/gateway/my.bucket") {
					t.Errorf("expected custom endpoint path and path-style bucket, got %s", r.URL.Path)
				}
				if !strings.Contains(r.Header.Get("Authorization"), "/garage/s3/aws4_request") {
					t.Errorf("request not signed for the configured region: %s", r.Header.Get("Authorization"))
				}
				if r.Header.Get("X-Amz-Checksum-Mode") != "" {
					t.Error("unexpected optional checksum validation request")
				}
				if r.Method == http.MethodPut {
					if r.Header.Get("Content-Encoding") == "aws-chunked" || r.Header.Get("X-Amz-Trailer") != "" || r.Header.Get("X-Amz-Sdk-Checksum-Algorithm") != "" {
						t.Error("upload unexpectedly uses optional AWS checksum framing")
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					if strings.HasSuffix(r.URL.Path, "/folder/") {
						if len(body) != 0 {
							t.Errorf("folder marker has body %q", body)
						}
					} else if string(body) != "test payload" || r.URL.Path != "/gateway/my.bucket/nested/a #?.txt" {
						t.Errorf("upload altered: path=%s, body=%q", r.URL.Path, body)
					}
					w.Header().Set("ETag", `"test-etag"`)
					return
				}
				if r.Method == http.MethodPost && r.URL.Query().Has("delete") {
					checksum := r.Header.Get("Content-Md5")
					for name := range r.Header {
						if strings.HasPrefix(strings.ToLower(name), "x-amz-checksum-") {
							checksum += r.Header.Get(name)
						}
					}
					if checksum == "" {
						t.Error("required bulk-delete checksum is missing")
					}
					w.Header().Set("Content-Type", "application/xml")
					io.WriteString(w, `<DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Deleted><Key>nested/a #?.txt</Key></Deleted></DeleteResult>`)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				io.WriteString(w, "test payload")
			}))
			if secure {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()
			client := newS3Client(server.URL+"/gateway", "garage", credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""))
			options := client.Options()
			options.HTTPClient = server.Client() // Trust only the test TLS certificate.
			client = s3.New(options)
			ctx := context.Background()
			for _, key := range []string{"nested/a #?.txt", "folder/"} {
				var body io.Reader
				var size int64
				if !strings.HasSuffix(key, "/") {
					body = strings.NewReader("test payload")
					size = 12
				}
				_, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String("my.bucket"), Key: aws.String(key), Body: body, ContentLength: aws.Int64(size)})
				if err != nil {
					t.Fatal(err)
				}
			}
			object, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String("my.bucket"), Key: aws.String("nested/a #?.txt")})
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(object.Body)
			object.Body.Close()
			if err != nil || string(body) != "test payload" {
				t.Fatalf("download: %q, %v", body, err)
			}
			_, err = client.DeleteObjects(ctx, &s3.DeleteObjectsInput{Bucket: aws.String("my.bucket"), Delete: &types.Delete{Objects: []types.ObjectIdentifier{{Key: aws.String("nested/a #?.txt")}}}})
			if err != nil {
				t.Fatal(err)
			}
			if requests.Load() != 4 {
				t.Fatalf("expected 4 requests, got %d", requests.Load())
			}
		})
	}
}
