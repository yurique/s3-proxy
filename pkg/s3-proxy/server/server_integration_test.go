// +build integration

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/stretchr/testify/assert"
)

func TestPublicRouter(t *testing.T) {
	trueValue := true
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tracingConfig := &config.TracingConfig{}

	type args struct {
		cfg *config.Config
	}
	tests := []struct {
		name               string
		args               args
		inputMethod        string
		inputURL           string
		inputBasicUser     string
		inputBasicPassword string
		inputBody          string
		inputFileName      string
		inputFileKey       string
		expectedCode       int
		expectedBody       string
		expectedHeaders    map[string]string
		notExpectedBody    string
		wantErr            bool
	}{
		{
			name: "GET a not found path",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/not-found/",
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "GET a folder without index document enabled",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/",
			expectedCode: 200,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET a folder without index document enabled and custom folder list template",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
							Templates: &config.TargetTemplateConfig{
								FolderList: &config.TargetTemplateConfigItem{
									InBucket: true,
									Path:     "templates/folder-list.tpl",
								},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/",
			expectedCode: 200,
			expectedBody: "fake template !",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET a file with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "GET a file with a not found error",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt-not-existing",
			expectedCode: 404,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Not Found /mount/folder1/test.txt-not-existing</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET a file with a not found error because of not valid host",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
								Host: "test.local",
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 404,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Not Found /mount/folder1/test.txt</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET a file with success on specific host",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
								Host: "test.local",
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://test.local/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "GET a file with forbidden error in case of no resource found",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								{
									Path:     "/mount/folder2/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 403,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET a file with forbidden error in case of no resource found because no valid http methods",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								{
									Path:     "/mount/folder2/*",
									Methods:  []string{"PUT"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 403,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET a file with unauthorized error in case of no basic auth",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control":    "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":     "text/html; charset=utf-8",
				"Www-Authenticate": "Basic realm=\"realm1\"",
			},
		},
		{
			name: "GET a file with unauthorized error in case of not found basic auth user",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/mount/folder1/test.txt",
			inputBasicUser:     "user2",
			inputBasicPassword: "pass2",
			expectedCode:       401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control":    "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":     "text/html; charset=utf-8",
				"Www-Authenticate": "Basic realm=\"realm1\"",
			},
		},
		{
			name: "GET a file with unauthorized error in case of wrong basic auth password",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/mount/folder1/test.txt",
			inputBasicUser:     "user1",
			inputBasicPassword: "pass2",
			expectedCode:       401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control":    "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":     "text/html; charset=utf-8",
				"Www-Authenticate": "Basic realm=\"realm1\"",
			},
		},
		{
			name: "GET a file with success in case of valid basic auth",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/mount/folder1/test.txt",
			inputBasicUser:     "user1",
			inputBasicPassword: "pass1",
			expectedCode:       200,
			expectedBody:       "Hello folder1!",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "GET a file with success in case of whitelist",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								{
									Path:      "/mount/folder1/test.txt",
									Methods:   []string{"GET"},
									WhiteList: &trueValue,
								},
								{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "GET target list",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{
						Enabled: true,
						Mount: &config.MountConfig{
							Path: []string{"/"},
						},
					},
					Tracing: tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/",
			expectedCode: 200,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Target buckets list</h1>
    <ul>
        <li>target1:
          <ul>
            <li><a href="/mount/">/mount/</a></li>
          </ul>
        </li>
    </ul>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET target list protected with basic authentication and without any password",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{
						Enabled: true,
						Mount: &config.MountConfig{
							Path: []string{"/"},
						},
						Resource: &config.Resource{
							Path:     "/*",
							Methods:  []string{"GET"},
							Provider: "provider1",
							Basic: &config.ResourceBasic{
								Credentials: []*config.BasicAuthUserConfig{
									{
										User: "user1",
										Password: &config.CredentialConfig{
											Value: "pass1",
										},
									},
								},
							},
						},
					},
					Tracing: tracingConfig,
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/",
			expectedCode: 401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control":    "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":     "text/html; charset=utf-8",
				"Www-Authenticate": "Basic realm=\"realm1\"",
			},
		},
		{
			name: "GET target list protected with basic authentication and with wrong user",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{
						Enabled: true,
						Mount: &config.MountConfig{
							Path: []string{"/"},
						},
						Resource: &config.Resource{
							Path:     "/*",
							Methods:  []string{"GET"},
							Provider: "provider1",
							Basic: &config.ResourceBasic{
								Credentials: []*config.BasicAuthUserConfig{
									{
										User: "user1",
										Password: &config.CredentialConfig{
											Value: "pass1",
										},
									},
								},
							},
						},
					},
					Tracing: tracingConfig,
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/",
			inputBasicUser:     "user2",
			inputBasicPassword: "pass1",
			expectedCode:       401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control":    "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":     "text/html; charset=utf-8",
				"Www-Authenticate": "Basic realm=\"realm1\"",
			},
		},
		{
			name: "GET target list protected with basic authentication and with wrong password",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{
						Enabled: true,
						Mount: &config.MountConfig{
							Path: []string{"/"},
						},
						Resource: &config.Resource{
							Path:     "/*",
							Methods:  []string{"GET"},
							Provider: "provider1",
							Basic: &config.ResourceBasic{
								Credentials: []*config.BasicAuthUserConfig{
									{
										User: "user1",
										Password: &config.CredentialConfig{
											Value: "pass1",
										},
									},
								},
							},
						},
					},
					Tracing: tracingConfig,
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/",
			inputBasicUser:     "user1",
			inputBasicPassword: "pass2",
			expectedCode:       401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control":    "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":     "text/html; charset=utf-8",
				"Www-Authenticate": "Basic realm=\"realm1\"",
			},
		},
		{
			name: "GET target list protected with basic authentication with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{
						Enabled: true,
						Mount: &config.MountConfig{
							Path: []string{"/"},
						},
						Resource: &config.Resource{
							Path:     "/*",
							Methods:  []string{"GET"},
							Provider: "provider1",
							Basic: &config.ResourceBasic{
								Credentials: []*config.BasicAuthUserConfig{
									{
										User: "user1",
										Password: &config.CredentialConfig{
											Value: "pass1",
										},
									},
								},
							},
						},
					},
					Tracing: tracingConfig,
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": {
								Realm: "realm1",
							},
						},
					},
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/",
			inputBasicUser:     "user1",
			inputBasicPassword: "pass1",
			expectedCode:       http.StatusOK,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Target buckets list</h1>
    <ul>
        <li>target1:
          <ul>
            <li><a href="/mount/">/mount/</a></li>
          </ul>
        </li>
    </ul>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET index document with index document enabled with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							IndexDocument: "index.html",
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/",
			expectedCode: 200,
			expectedBody: "<!DOCTYPE html><html><body><h1>Hello folder1!</h1></body></html>",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "GET a folder path with index document enabled and index document not found with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							IndexDocument: "index.html-fake",
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:     "GET",
			inputURL:        "http://localhost/mount/folder1/",
			expectedCode:    200,
			notExpectedBody: "<!DOCTYPE html><html><body><h1>Hello folder1!</h1></body></html>",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "DELETE a path with a 405 error (method not allowed) because DELETE not enabled",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "DELETE",
			inputURL:     "http://localhost/mount/folder1/text.txt",
			expectedCode: http.StatusMethodNotAllowed,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
			},
		},
		{
			name: "DELETE a path with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET:    &config.GetActionConfig{Enabled: true},
								DELETE: &config.DeleteActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "DELETE",
			inputURL:     "http://localhost/mount/folder1/text.txt",
			expectedCode: 204,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
			},
		},
		{
			name: "PUT in a path with success without allow override and don't need it",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
									Config: &config.PutActionConfigConfig{
										StorageClass: "Standard",
										Metadata: map[string]string{
											"meta1": "meta1",
										},
									},
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test2.txt",
			inputFileKey:  "file",
			inputBody:     "Hello test2!",
			expectedCode:  204,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
			},
		},
		{
			name: "PUT in a path without allow override should failed",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
									Config: &config.PutActionConfigConfig{
										StorageClass: "Standard",
										Metadata: map[string]string{
											"meta1": "meta1",
										},
									},
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test.txt",
			inputFileKey:  "file",
			inputBody:     "Hello test1!",
			expectedCode:  403,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "PUT in a path with allow override should be ok",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
									Config: &config.PutActionConfigConfig{
										StorageClass: "Standard",
										Metadata: map[string]string{
											"meta1": "meta1",
										},
										AllowOverride: true,
									},
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test.txt",
			inputFileKey:  "file",
			inputBody:     "Hello test1!",
			expectedCode:  204,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
			},
		},
		{
			name: "PUT in a path should fail because no input",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
								},
							},
						},
					},
				},
			},
			inputMethod:  "PUT",
			inputURL:     "http://localhost/mount/folder1/",
			expectedCode: 500,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>missing form body</p>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "PUT in a path should fail because wrong key in form",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates: &config.TemplateConfig{
						FolderList:          "../../../templates/folder-list.tpl",
						TargetList:          "../../../templates/target-list.tpl",
						NotFound:            "../../../templates/not-found.tpl",
						Forbidden:           "../../../templates/forbidden.tpl",
						BadRequest:          "../../../templates/bad-request.tpl",
						InternalServerError: "../../../templates/internal-server-error.tpl",
						Unauthorized:        "../../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						{
							Name: "target1",
							Bucket: &config.BucketConfig{
								Name:       bucket,
								Region:     region,
								S3Endpoint: s3server.URL,
								Credentials: &config.BucketCredentialConfig{
									AccessKey: &config.CredentialConfig{Value: accessKey},
									SecretKey: &config.CredentialConfig{Value: secretAccessKey},
								},
								DisableSSL: true,
							},
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test.txt",
			inputFileKey:  "wrongkey",
			inputBody:     "Hello test1!",
			expectedCode:  500,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>http: no such file</p>
  </body>
</html>
`,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(tt.args.cfg)

			logger := log.NewLogger()
			// Create tracing service
			tsvc, err := tracing.New(cfgManagerMock, logger)
			assert.NoError(t, err)

			svr := &Server{
				logger:     logger,
				cfgManager: cfgManagerMock,
				metricsCl:  metricsCtx,
				tracingSvc: tsvc,
			}
			got, err := svr.generateRouter()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// If want error at this moment => stop
			if tt.wantErr {
				return
			}
			w := httptest.NewRecorder()
			req, err := http.NewRequest(
				tt.inputMethod,
				tt.inputURL,
				nil,
			)
			if err != nil {
				t.Error(err)
				return
			}
			// multipart form
			if tt.inputBody != "" {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile(tt.inputFileKey, filepath.Base(tt.inputFileName))
				if err != nil {
					t.Error(err)
					return
				}
				_, err = io.Copy(part, strings.NewReader(tt.inputBody))
				if err != nil {
					t.Error(err)
					return
				}
				err = writer.Close()
				if err != nil {
					t.Error(err)
					return
				}
				req, err = http.NewRequest(
					tt.inputMethod,
					tt.inputURL,
					body,
				)
				if err != nil {
					t.Error(err)
					return
				}
				req.Header.Set("Content-Type", writer.FormDataContentType())
			}
			// Add basic auth
			if tt.inputBasicUser != "" {
				req.SetBasicAuth(tt.inputBasicUser, tt.inputBasicPassword)
			}
			got.ServeHTTP(w, req)

			if tt.expectedBody != "" {
				body := w.Body.String()
				if tt.expectedBody != body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", expected body \"%v\"", body, tt.expectedBody)
				}
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				if tt.notExpectedBody == body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", not expected body \"%v\"", body, tt.notExpectedBody)
				}
			}

			if tt.expectedHeaders != nil {
				for key, val := range tt.expectedHeaders {
					wheader := w.HeaderMap.Get(key)
					if val != wheader {
						t.Errorf("Integration test on GenerateRouter() header %s = %v, expected %v", key, wheader, val)
					}
				}
			}

			if tt.expectedCode != w.Code {
				t.Errorf("Integration test on GenerateRouter() status code = %v, expected status code %v", w.Code, tt.expectedCode)
			}
		})
	}
}

func TestTracing(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplConfig := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	targetsCfg := []*config.TargetConfig{
		{
			Name: "target1",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/mount/"},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
	}
	type args struct {
		cfg *config.Config
	}
	tests := []struct {
		name               string
		args               args
		inputMethod        string
		inputURL           string
		inputBasicUser     string
		inputBasicPassword string
		inputBody          string
		inputFileName      string
		inputFileKey       string
		expectedCode       int
		expectedBody       string
		expectedHeaders    map[string]string
		notExpectedBody    string
		wantErr            bool
	}{
		{
			name: "GET a not found path without any tracing configuration",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/not-found/",
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "GET a not found path with a tracing configuration",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Tracing: &config.TracingConfig{
						Enabled:       true,
						UDPHost:       "localhost:6831",
						FlushInterval: "120s",
					},
					Templates: tplConfig,
					Targets:   targetsCfg,
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/not-found/",
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(tt.args.cfg)

			logger := log.NewLogger()
			// Create tracing service
			tsvc, err := tracing.New(cfgManagerMock, logger)
			assert.NoError(t, err)

			svr := &Server{
				logger:     logger,
				cfgManager: cfgManagerMock,
				metricsCl:  metricsCtx,
				tracingSvc: tsvc,
			}
			got, err := svr.generateRouter()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// If want error at this moment => stop
			if tt.wantErr {
				return
			}
			w := httptest.NewRecorder()
			req, err := http.NewRequest(
				tt.inputMethod,
				tt.inputURL,
				nil,
			)
			if err != nil {
				t.Error(err)
				return
			}
			// multipart form
			if tt.inputBody != "" {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile(tt.inputFileKey, filepath.Base(tt.inputFileName))
				if err != nil {
					t.Error(err)
					return
				}
				_, err = io.Copy(part, strings.NewReader(tt.inputBody))
				if err != nil {
					t.Error(err)
					return
				}
				err = writer.Close()
				if err != nil {
					t.Error(err)
					return
				}
				req, err = http.NewRequest(
					tt.inputMethod,
					tt.inputURL,
					body,
				)
				if err != nil {
					t.Error(err)
					return
				}
				req.Header.Set("Content-Type", writer.FormDataContentType())
			}
			// Add basic auth
			if tt.inputBasicUser != "" {
				req.SetBasicAuth(tt.inputBasicUser, tt.inputBasicPassword)
			}
			got.ServeHTTP(w, req)

			if tt.expectedBody != "" {
				body := w.Body.String()
				if tt.expectedBody != body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", expected body \"%v\"", body, tt.expectedBody)
				}
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				if tt.notExpectedBody == body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", not expected body \"%v\"", body, tt.notExpectedBody)
				}
			}

			if tt.expectedHeaders != nil {
				for key, val := range tt.expectedHeaders {
					wheader := w.HeaderMap.Get(key)
					if val != wheader {
						t.Errorf("Integration test on GenerateRouter() header %s = %v, expected %v", key, wheader, val)
					}
				}
			}

			if tt.expectedCode != w.Code {
				t.Errorf("Integration test on GenerateRouter() status code = %v, expected status code %v", w.Code, tt.expectedCode)
			}
		})
	}
}

// This is in a separate test because this one will need a real server to discuss with OIDC server
func TestOIDCAuthentication(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"

	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplCfg := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	tracingConfig := &config.TracingConfig{}
	targetsTpl := []*config.TargetConfig{
		{
			Name: "target1",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/mount/"},
			},
			Resources: []*config.Resource{
				{
					Path:     "/mount/folder1/*",
					Methods:  []string{"GET"},
					Provider: "provider1",
					OIDC: &config.ResourceOIDC{
						AuthorizationAccesses: []*config.OIDCAuthorizationAccess{},
					},
				},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
		{
			Name: "target-multiple-providers",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/mount-multiple-provider/"},
			},
			Resources: []*config.Resource{
				{
					Path:     "/mount-multiple-provider/folder1/*",
					Methods:  []string{"GET"},
					Provider: "provider2",
					OIDC: &config.ResourceOIDC{
						AuthorizationAccesses: []*config.OIDCAuthorizationAccess{},
					},
				},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
		{
			Name: "target-opa-server",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/mount-opa-server/"},
			},
			Resources: []*config.Resource{
				{
					Path:     "/mount-opa-server/folder1/*",
					Methods:  []string{"GET"},
					Provider: "provider1",
					OIDC: &config.ResourceOIDC{
						AuthorizationOPAServer: &config.OPAServerAuthorization{
							URL: "http://localhost:8181/v1/data/example/authz/allowed",
						},
					},
				},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
		{
			Name: "target-wrong-opa-server-url",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/mount-wrong-opa-server-url/"},
			},
			Resources: []*config.Resource{
				{
					Path:     "/mount-wrong-opa-server-url/folder1/*",
					Methods:  []string{"GET"},
					Provider: "provider1",
					OIDC: &config.ResourceOIDC{
						AuthorizationOPAServer: &config.OPAServerAuthorization{
							URL: "http://fake.fake",
						},
					},
				},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
		{
			Name: "target-with-group",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/mount-with-group/"},
			},
			Resources: []*config.Resource{
				{
					Path:     "/mount-with-group/folder1/*",
					Methods:  []string{"GET"},
					Provider: "provider1",
					OIDC: &config.ResourceOIDC{
						AuthorizationAccesses: []*config.OIDCAuthorizationAccess{
							&config.OIDCAuthorizationAccess{
								Group: "group1",
							},
						},
					},
				},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
	}
	svrCfg := &config.ServerConfig{
		ListenAddr: "",
		Port:       8080,
	}

	type jwtToken struct {
		IDToken string `json:"id_token"`
	}
	type args struct {
		cfg *config.Config
	}
	tests := []struct {
		name                              string
		args                              args
		inputURL                          string
		inputForgeOIDCHeader              bool
		inputForgeOIDCUsername            string
		inputForgeOIDCPassword            string
		inputForgeOIDCWithoutClientSecret bool
		expectedCode                      int
		expectedBody                      string
		expectedResponseHost              string
		expectedResponsePath              string
		expectedHeaders                   map[string]string
		notExpectedBody                   string
		wantErr                           bool
	}{
		{
			name: "Inject not working OIDC provider",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:     "fake-client-id",
								CookieName:   "oidc",
								RedirectURL:  "http://fake-s3-proxy/",
								CallbackPath: "/auth/provider1/callback",
								IssuerURL:    "https://fake-idp/",
								LoginPath:    "/auth/provider1/",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			wantErr: true,
		},
		{
			name: "GET a file with redirect to oidc provider in case of no oidc cookie or bearer token",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:     "client-with-secret",
								ClientSecret: &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:   "oidc",
								RedirectURL:  "http://localhost:8080/",
								CallbackPath: "/auth/provider1/callback",
								IssuerURL:    "http://localhost:8088/auth/realms/integration",
								LoginPath:    "/auth/provider1/",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL: "http://localhost:8080/mount/folder1/test.txt",
			wantErr:  false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-store, must-revalidate, max-age=0",
				"Content-Type":  "text/html;charset=utf-8",
			},
			expectedResponseHost: "localhost:8088",
			expectedResponsePath: "/auth/realms/integration/protocol/openid-connect/auth",
			expectedCode:         200,
		},
		{
			name: "GET a file with oidc bearer token should be ok",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:     "client-with-secret",
								ClientSecret: &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:   "oidc",
								RedirectURL:  "http://localhost:8080/",
								CallbackPath: "/auth/provider1/callback",
								IssuerURL:    "http://localhost:8088/auth/realms/integration",
								LoginPath:    "/auth/provider1/",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount/folder1/test.txt",
			expectedCode:         200,
		},
		{
			name: "GET a file with oidc bearer token and email verified flag enabled should be ok",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:      "client-with-secret",
								ClientSecret:  &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:    "oidc",
								RedirectURL:   "http://localhost:8080/",
								CallbackPath:  "/auth/provider1/callback",
								IssuerURL:     "http://localhost:8088/auth/realms/integration",
								LoginPath:     "/auth/provider1/",
								EmailVerified: true,
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount/folder1/test.txt",
			expectedCode:         200,
		},
		{
			name: "GET a file with oidc bearer token and email verified flag enabled should be forbidden",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:      "client-with-secret",
								ClientSecret:  &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:    "oidc",
								RedirectURL:   "http://localhost:8080/",
								CallbackPath:  "/auth/provider1/callback",
								IssuerURL:     "http://localhost:8088/auth/realms/integration",
								LoginPath:     "/auth/provider1/",
								EmailVerified: true,
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user-not-verified",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount/folder1/test.txt",
			expectedCode:         403,
		},
		{
			name: "GET a file with oidc bearer token and group authorization enabled should be forbidden",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:      "client-with-secret",
								ClientSecret:  &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:    "oidc",
								RedirectURL:   "http://localhost:8080/",
								CallbackPath:  "/auth/provider1/callback",
								IssuerURL:     "http://localhost:8088/auth/realms/integration",
								LoginPath:     "/auth/provider1/",
								EmailVerified: true,
								GroupClaim:    "groups",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount-with-group/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user-without-group",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount-with-group/folder1/test.txt",
			expectedCode:         403,
		},
		{
			name: "GET a file with oidc bearer token and group authorization enabled should be ok",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:      "client-with-secret",
								ClientSecret:  &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:    "oidc",
								RedirectURL:   "http://localhost:8080/",
								CallbackPath:  "/auth/provider1/callback",
								IssuerURL:     "http://localhost:8088/auth/realms/integration",
								LoginPath:     "/auth/provider1/",
								EmailVerified: true,
								GroupClaim:    "groups",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount-with-group/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount-with-group/folder1/test.txt",
			expectedCode:         200,
		},
		{
			name: "GET a file with oidc bearer token and opa server enabled should be forbidden",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:      "client-with-secret",
								ClientSecret:  &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:    "oidc",
								RedirectURL:   "http://localhost:8080/",
								CallbackPath:  "/auth/provider1/callback",
								IssuerURL:     "http://localhost:8088/auth/realms/integration",
								LoginPath:     "/auth/provider1/",
								EmailVerified: true,
								GroupClaim:    "groups",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount-opa-server/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user-without-group",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount-opa-server/folder1/test.txt",
			expectedCode:         403,
		},
		{
			name: "GET a file with oidc bearer token and opa server enabled should be ok",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:      "client-with-secret",
								ClientSecret:  &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:    "oidc",
								RedirectURL:   "http://localhost:8080/",
								CallbackPath:  "/auth/provider1/callback",
								IssuerURL:     "http://localhost:8088/auth/realms/integration",
								LoginPath:     "/auth/provider1/",
								EmailVerified: true,
								GroupClaim:    "groups",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount-opa-server/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount-opa-server/folder1/test.txt",
			expectedCode:         200,
		},
		{
			name: "GET a file with oidc bearer token and opa server enabled should be forbidden",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:      "client-with-secret",
								ClientSecret:  &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:    "oidc",
								RedirectURL:   "http://localhost:8080/",
								CallbackPath:  "/auth/provider1/callback",
								IssuerURL:     "http://localhost:8088/auth/realms/integration",
								LoginPath:     "/auth/provider1/",
								EmailVerified: true,
								GroupClaim:    "groups",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:               "http://localhost:8080/mount-wrong-opa-server-url/folder1/test.txt",
			inputForgeOIDCHeader:   true,
			inputForgeOIDCUsername: "user-without-group",
			inputForgeOIDCPassword: "password",
			wantErr:                false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount-wrong-opa-server-url/folder1/test.txt",
			expectedCode:         500,
		},
		{
			name: "GET a file with oidc bearer token should be ok (with multiple providers)",
			args: args{
				cfg: &config.Config{
					Server:      svrCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     tracingConfig,
					Templates:   tplCfg,
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": {
								ClientID:     "client-with-secret",
								ClientSecret: &config.CredentialConfig{Value: "565f78f2-a706-41cd-a1a0-431d7df29443"},
								CookieName:   "oidc",
								RedirectURL:  "http://localhost:8080/",
								CallbackPath: "/auth/provider1/callback",
								IssuerURL:    "http://localhost:8088/auth/realms/integration",
								LoginPath:    "/auth/provider1/",
							},
							"provider2": {
								ClientID:     "client-without-secret",
								ClientSecret: nil,
								CookieName:   "oidc",
								RedirectURL:  "http://localhost:8080/",
								CallbackPath: "/auth/provider2/callback",
								IssuerURL:    "http://localhost:8088/auth/realms/integration",
								LoginPath:    "/auth/provider2/",
							},
						},
					},
					Targets: targetsTpl,
				},
			},
			inputURL:                          "http://localhost:8080/mount-multiple-provider/folder1/test.txt",
			inputForgeOIDCHeader:              true,
			inputForgeOIDCUsername:            "user",
			inputForgeOIDCPassword:            "password",
			inputForgeOIDCWithoutClientSecret: true,
			wantErr:                           false,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
			expectedResponseHost: "localhost:8080",
			expectedResponsePath: "/mount-multiple-provider/folder1/test.txt",
			expectedCode:         200,
			expectedBody:         "Hello folder1!",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(tt.args.cfg)
			cfgManagerMock.EXPECT().AddOnChangeHook(gomock.Any()).AnyTimes()

			logger := log.NewLogger()
			// Create tracing service
			tsvc, err := tracing.New(cfgManagerMock, logger)
			assert.NoError(t, err)

			ssvr := NewServer(logger, cfgManagerMock, metricsCtx, tsvc)
			err = ssvr.GenerateServer()
			if (err != nil) != tt.wantErr {
				t.Errorf("generateServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// If want error at this moment => stop
			if tt.wantErr {
				return
			}

			var wg sync.WaitGroup

			// Add a wait
			wg.Add(1)
			// Listen and synchronize wait
			go func() error {
				wg.Done()
				err := ssvr.Listen()

				if err != http.ErrServerClosed {
					assert.NoError(t, err)
					return err
				}

				return nil
			}()
			// Wait server up and running
			wg.Wait()
			// Force a sleep in order to wait server up and running
			time.Sleep(time.Second)
			// Defer close server
			defer func() {
				err := ssvr.server.Close()
				assert.NoError(t, err)
			}()

			request, err := http.NewRequest("GET", tt.inputURL, nil) // URL-encoded payload
			// Check err
			if err != nil {
				t.Error(err)
				return
			}

			if tt.inputForgeOIDCHeader {
				data := url.Values{}
				data.Set("username", tt.inputForgeOIDCUsername)
				data.Set("password", tt.inputForgeOIDCPassword)
				if tt.inputForgeOIDCWithoutClientSecret {
					data.Set("client_id", "client-without-secret")
				} else {
					data.Set("client_id", "client-with-secret")
					data.Set("client_secret", "565f78f2-a706-41cd-a1a0-431d7df29443")
				}
				data.Set("grant_type", "password")
				data.Set("scope", "openid profile email")

				authentUrlStr := "http://localhost:8088/auth/realms/integration/protocol/openid-connect/token"

				clientAuth := &http.Client{}
				r, err := http.NewRequest("POST", authentUrlStr, strings.NewReader(data.Encode())) // URL-encoded payload
				// Check err
				if err != nil {
					t.Error(err)
					return
				}

				r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

				resp, err := clientAuth.Do(r)
				// Check err
				if err != nil {
					t.Error(err)
					return
				}

				bodyBytes, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					t.Error(err)
					return
				}
				body := string(bodyBytes)

				// Check response
				if resp.StatusCode != 200 {
					t.Errorf("%d - %s", resp.StatusCode, body)
					return
				}

				var to jwtToken
				// Parse token
				err = json.Unmarshal(bodyBytes, &to)
				if err != nil {
					t.Error(err)
					return
				}

				// Add header to request
				request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", to.IDToken))
			}

			// Create http client
			client := &http.Client{
				Timeout: 10 * time.Second,
			}
			resp, err := client.Do(request)

			if err != nil {
				t.Error(err)
				return
			}

			if tt.expectedResponseHost != resp.Request.URL.Host {
				t.Errorf("OIDC Integration test on GenerateRouter() response host = %v, expected response host %v", resp.Request.URL.Host, tt.expectedResponseHost)
			}

			if tt.expectedResponsePath != resp.Request.URL.Path {
				t.Errorf("OIDC Integration test on GenerateRouter() response path = %v, expected response path %v", resp.Request.URL.Path, tt.expectedResponsePath)
			}

			if tt.expectedHeaders != nil {
				for key, val := range tt.expectedHeaders {
					wheader := resp.Header.Get(key)
					if val != wheader {
						t.Errorf("OIDC Integration test on GenerateRouter() header %s = %v, expected %v", key, wheader, val)
					}
				}
			}

			if tt.expectedCode != resp.StatusCode {
				t.Errorf("OIDC Integration test on GenerateRouter() status code = %v, expected status code %v", resp.StatusCode, tt.expectedCode)
			}

			if tt.expectedBody != "" {
				bodyBytes, _ := ioutil.ReadAll(resp.Body)
				body := string(bodyBytes)
				if tt.expectedBody != body {
					t.Errorf("OIDC Integration test on GenerateRouter() body = \"%v\", not expected body \"%v\"", body, tt.expectedBody)
				}
			}
		})
	}
}

func TestCORS(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplConfig := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	targetsCfg := []*config.TargetConfig{
		{
			Name: "target1",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/mount/"},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
	}
	type args struct {
		cfg *config.Config
	}
	tests := []struct {
		name            string
		args            args
		inputMethod     string
		inputURL        string
		inputHeaders    map[string]string
		expectedCode    int
		expectedBody    string
		expectedHeaders map[string]string
		notExpectedBody string
		wantErr         bool
	}{
		{
			name: "CORS disabled",
			args: args{
				cfg: &config.Config{
					Server: &config.ServerConfig{
						Port: 8080,
						CORS: &config.ServerCorsConfig{
							Enabled: false,
						},
					},
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "CORS allow all",
			args: args{
				cfg: &config.Config{
					Server: &config.ServerConfig{
						Port: 8080,
						CORS: &config.ServerCorsConfig{
							Enabled:  true,
							AllowAll: true,
						},
					},
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control":               "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":                "text/plain; charset=utf-8",
				"Access-Control-Allow-Origin": "*",
			},
		},
		{
			name: "CORS enabled with fixed domain (allowed)",
			args: args{
				cfg: &config.Config{
					Server: &config.ServerConfig{
						Port: 8080,
						CORS: &config.ServerCorsConfig{
							Enabled:      true,
							AllowAll:     false,
							AllowMethods: []string{"GET"},
							AllowOrigins: []string{"https://test.fake"},
						},
					},
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control":               "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":                "text/plain; charset=utf-8",
				"Access-Control-Allow-Origin": "https://test.fake",
			},
		},
		{
			name: "CORS enabled with fixed domain (not allowed)",
			args: args{
				cfg: &config.Config{
					Server: &config.ServerConfig{
						Port: 8080,
						CORS: &config.ServerCorsConfig{
							Enabled:      true,
							AllowAll:     false,
							AllowMethods: []string{"GET"},
							AllowOrigins: []string{"https://test.fake"},
						},
					},
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test.test",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control":               "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":                "text/plain; charset=utf-8",
				"Access-Control-Allow-Origin": "",
			},
		},
		{
			name: "CORS enabled with star domain (allowed)",
			args: args{
				cfg: &config.Config{
					Server: &config.ServerConfig{
						Port: 8080,
						CORS: &config.ServerCorsConfig{
							Enabled:      true,
							AllowAll:     false,
							AllowMethods: []string{"GET"},
							AllowOrigins: []string{"https://test.*"},
						},
					},
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control":               "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":                "text/plain; charset=utf-8",
				"Access-Control-Allow-Origin": "https://test.fake",
			},
		},
		{
			name: "CORS enabled with star domain (not allowed)",
			args: args{
				cfg: &config.Config{
					Server: &config.ServerConfig{
						Port: 8080,
						CORS: &config.ServerCorsConfig{
							Enabled:      true,
							AllowAll:     false,
							AllowMethods: []string{"GET"},
							AllowOrigins: []string{"https://test.*"},
						},
					},
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test2.test",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control":               "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":                "text/plain; charset=utf-8",
				"Access-Control-Allow-Origin": "",
			},
		},
		{
			name: "CORS enabled with not allowed method",
			args: args{
				cfg: &config.Config{
					Server: &config.ServerConfig{
						Port: 8080,
						CORS: &config.ServerCorsConfig{
							Enabled:      true,
							AllowAll:     false,
							AllowMethods: []string{"DELETE"},
							AllowOrigins: []string{"https://test.*"},
						},
					},
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets:     targetsCfg,
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test.test",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedBody: "404 page not found\n",
			expectedHeaders: map[string]string{
				"Cache-Control":               "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":                "text/plain; charset=utf-8",
				"Access-Control-Allow-Origin": "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(tt.args.cfg)

			logger := log.NewLogger()
			// Create tracing service
			tsvc, err := tracing.New(cfgManagerMock, logger)
			assert.NoError(t, err)

			svr := &Server{
				logger:     logger,
				cfgManager: cfgManagerMock,
				metricsCl:  metricsCtx,
				tracingSvc: tsvc,
			}
			got, err := svr.generateRouter()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// If want error at this moment => stop
			if tt.wantErr {
				return
			}
			w := httptest.NewRecorder()
			req, err := http.NewRequest(
				tt.inputMethod,
				tt.inputURL,
				nil,
			)
			if err != nil {
				t.Error(err)
				return
			}

			// Set input headers
			if tt.inputHeaders != nil {
				for k, v := range tt.inputHeaders {
					req.Header.Set(k, v)
				}
			}

			got.ServeHTTP(w, req)

			if tt.expectedBody != "" {
				body := w.Body.String()
				if tt.expectedBody != body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", expected body \"%v\"", body, tt.expectedBody)
				}
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				if tt.notExpectedBody == body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", not expected body \"%v\"", body, tt.notExpectedBody)
				}
			}

			if tt.expectedHeaders != nil {
				for key, val := range tt.expectedHeaders {
					wheader := w.HeaderMap.Get(key)
					if val != wheader {
						t.Errorf("Integration test on GenerateRouter() header %s = %v, expected %v", key, wheader, val)
					}
				}
			}

			if tt.expectedCode != w.Code {
				t.Errorf("Integration test on GenerateRouter() status code = %v, expected status code %v", w.Code, tt.expectedCode)
			}
		})
	}
}

func TestIndexLargeBucket(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplConfig := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	targetsCfg := []*config.TargetConfig{
		{
			Name: "target1",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL:    true,
				S3ListMaxKeys: 1000,
			},
			Mount: &config.MountConfig{
				Path: []string{"/"},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
			IndexDocument: "index.html",
		},
	}

	// Create go mock controller
	ctrl := gomock.NewController(t)
	cfgManagerMock := cmocks.NewMockManager(ctrl)

	// Load configuration in manager
	cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
		Server: &config.ServerConfig{
			Port: 8080,
		},
		ListTargets: &config.ListTargetsConfig{},
		Tracing:     &config.TracingConfig{},
		Templates:   tplConfig,
		Targets:     targetsCfg,
	})

	logger := log.NewLogger()
	// Create tracing service
	tsvc, err := tracing.New(cfgManagerMock, logger)
	assert.NoError(t, err)

	svr := &Server{
		logger:     logger,
		cfgManager: cfgManagerMock,
		metricsCl:  metricsCtx,
		tracingSvc: tsvc,
	}
	got, err := svr.generateRouter()
	if err != nil {
		t.Errorf("TestIndexLargeBucket.GenerateRouter() error = %v", err)
		return
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest(
		"GET",
		"http://localhost/folder3/",
		nil,
	)
	if err != nil {
		t.Error(err)
		return
	}

	got.ServeHTTP(w, req)

	// Test status code
	if w.Code != 200 {
		t.Errorf("TestIndexLargeBucket.GenerateRouter() status code = %v, expected status code %v", w.Code, 200)
	}
	// Test body
	body := w.Body.String()
	expectedBody := "<!DOCTYPE html><html><body><h1>Hello folder3!</h1></body></html>"
	if body != expectedBody {
		t.Errorf("TestIndexLargeBucket.GenerateRouter() body = \"%v\", expected body \"%v\"", body, expectedBody)
	}
}

func TestListLargeBucketAndSmallMaxKeys(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	maxKeys := 500
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplConfig := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	targetsCfg := []*config.TargetConfig{
		{
			Name: "target1",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL:    true,
				S3ListMaxKeys: int64(maxKeys),
			},
			Mount: &config.MountConfig{
				Path: []string{"/"},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
	}

	// Create go mock controller
	ctrl := gomock.NewController(t)
	cfgManagerMock := cmocks.NewMockManager(ctrl)

	// Load configuration in manager
	cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
		Server: &config.ServerConfig{
			Port: 8080,
		},
		ListTargets: &config.ListTargetsConfig{},
		Tracing:     &config.TracingConfig{},
		Templates:   tplConfig,
		Targets:     targetsCfg,
	})

	logger := log.NewLogger()
	// Create tracing service
	tsvc, err := tracing.New(cfgManagerMock, logger)
	assert.NoError(t, err)

	svr := &Server{
		logger:     logger,
		cfgManager: cfgManagerMock,
		metricsCl:  metricsCtx,
		tracingSvc: tsvc,
	}
	got, err := svr.generateRouter()
	if err != nil {
		t.Errorf("TestListLargeBucketAndSmallMaxKeys.GenerateRouter() error = %v", err)
		return
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest(
		"GET",
		"http://localhost/folder3/",
		nil,
	)
	if err != nil {
		t.Error(err)
		return
	}

	got.ServeHTTP(w, req)

	// Test status code
	if w.Code != 200 {
		t.Errorf("TestListLargeBucketAndSmallMaxKeys.GenerateRouter() status code = %v, expected status code %v", w.Code, 200)
	}
	// Test body
	body := w.Body.String()
	if strings.Count(body, "\"/folder3/") != maxKeys {
		t.Errorf("TestListLargeBucketAndSmallMaxKeys.GenerateRouter() body = \"%v\", must contains %d elements in the list", body, maxKeys)
	}
}

func TestListLargeBucketAndMaxKeysGreaterThanS3MaxKeys(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	maxKeys := 1500
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplConfig := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	targetsCfg := []*config.TargetConfig{
		{
			Name: "target1",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL:    true,
				S3ListMaxKeys: int64(maxKeys),
			},
			Mount: &config.MountConfig{
				Path: []string{"/"},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
	}

	// Create go mock controller
	ctrl := gomock.NewController(t)
	cfgManagerMock := cmocks.NewMockManager(ctrl)

	// Load configuration in manager
	cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
		Server: &config.ServerConfig{
			Port: 8080,
		},
		ListTargets: &config.ListTargetsConfig{},
		Tracing:     &config.TracingConfig{},
		Templates:   tplConfig,
		Targets:     targetsCfg,
	})

	logger := log.NewLogger()
	// Create tracing service
	tsvc, err := tracing.New(cfgManagerMock, logger)
	assert.NoError(t, err)

	svr := &Server{
		logger:     logger,
		cfgManager: cfgManagerMock,
		metricsCl:  metricsCtx,
		tracingSvc: tsvc,
	}
	got, err := svr.generateRouter()
	if err != nil {
		t.Errorf("TestListLargeBucketAndMaxKeysGreaterThanS3MaxKeys.GenerateRouter() error = %v", err)
		return
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest(
		"GET",
		"http://localhost/folder3/",
		nil,
	)
	if err != nil {
		t.Error(err)
		return
	}

	got.ServeHTTP(w, req)

	// Test status code
	if w.Code != 200 {
		t.Errorf("TestListLargeBucketAndMaxKeysGreaterThanS3MaxKeys.GenerateRouter() status code = %v, expected status code %v", w.Code, 200)
	}
	// Test body
	body := w.Body.String()
	if strings.Count(body, "\"/folder3/") != maxKeys {
		t.Errorf("TestListLargeBucketAndMaxKeysGreaterThanS3MaxKeys.GenerateRouter() body = \"%v\", must contains %d elements in the list", body, maxKeys)
	}
}

func TestFolderWithSubFolders(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplConfig := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	targetsCfg := []*config.TargetConfig{
		{
			Name: "target1",
			Bucket: &config.BucketConfig{
				Name:       bucket,
				Region:     region,
				S3Endpoint: s3server.URL,
				Credentials: &config.BucketCredentialConfig{
					AccessKey: &config.CredentialConfig{Value: accessKey},
					SecretKey: &config.CredentialConfig{Value: secretAccessKey},
				},
				DisableSSL: true,
			},
			Mount: &config.MountConfig{
				Path: []string{"/"},
			},
			Actions: &config.ActionsConfig{
				GET: &config.GetActionConfig{Enabled: true},
			},
		},
	}

	// Create go mock controller
	ctrl := gomock.NewController(t)
	cfgManagerMock := cmocks.NewMockManager(ctrl)

	// Load configuration in manager
	cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
		Server: &config.ServerConfig{
			Port: 8080,
		},
		ListTargets: &config.ListTargetsConfig{},
		Tracing:     &config.TracingConfig{},
		Templates:   tplConfig,
		Targets:     targetsCfg,
	})

	logger := log.NewLogger()
	// Create tracing service
	tsvc, err := tracing.New(cfgManagerMock, logger)
	assert.NoError(t, err)

	svr := &Server{
		logger:     logger,
		cfgManager: cfgManagerMock,
		metricsCl:  metricsCtx,
		tracingSvc: tsvc,
	}
	got, err := svr.generateRouter()
	if err != nil {
		t.Errorf("TestFolderWithSubFolders.GenerateRouter() error = %v", err)
		return
	}
	w := httptest.NewRecorder()
	req, err := http.NewRequest(
		"GET",
		"http://localhost/folder4/",
		nil,
	)
	if err != nil {
		t.Error(err)
		return
	}

	got.ServeHTTP(w, req)

	// Test status code
	if w.Code != 200 {
		t.Errorf("TestFolderWithSubFolders.GenerateRouter() status code = %v, expected status code %v", w.Code, 200)
	}
	// Test body
	body := w.Body.String()
	if !strings.Contains(body, "\"/folder4/test.txt") {
		t.Errorf("TestFolderWithSubFolders.GenerateRouter() body = \"%v\", must contains text.txt file", body)
	}
	if !strings.Contains(body, "\"/folder4/index.html") {
		t.Errorf("TestFolderWithSubFolders.GenerateRouter() body = \"%v\", must contains index.html file", body)
	}
	if !strings.Contains(body, "\"/folder4/sub1/") {
		t.Errorf("TestFolderWithSubFolders.GenerateRouter() body = \"%v\", must contains sub1 folder", body)
	}
	if !strings.Contains(body, "\"/folder4/sub2/") {
		t.Errorf("TestFolderWithSubFolders.GenerateRouter() body = \"%v\", must contains sub2 folder", body)
	}
}

func TestTrailingSlashRedirect(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	tplConfig := &config.TemplateConfig{
		FolderList:          "../../../templates/folder-list.tpl",
		TargetList:          "../../../templates/target-list.tpl",
		NotFound:            "../../../templates/not-found.tpl",
		Forbidden:           "../../../templates/forbidden.tpl",
		BadRequest:          "../../../templates/bad-request.tpl",
		InternalServerError: "../../../templates/internal-server-error.tpl",
		Unauthorized:        "../../../templates/unauthorized.tpl",
	}
	srvCfg := &config.ServerConfig{
		Port: 8080,
	}
	bucketCfg := &config.BucketConfig{
		Name:       bucket,
		Region:     region,
		S3Endpoint: s3server.URL,
		Credentials: &config.BucketCredentialConfig{
			AccessKey: &config.CredentialConfig{Value: accessKey},
			SecretKey: &config.CredentialConfig{Value: secretAccessKey},
		},
		DisableSSL: true,
	}
	type args struct {
		cfg *config.Config
	}
	tests := []struct {
		name            string
		args            args
		inputMethod     string
		inputURL        string
		inputHeaders    map[string]string
		expectedCode    int
		expectedBody    string
		expectedHeaders map[string]string
		notExpectedBody string
		wantErr         bool
	}{
		{
			name: "Don't force trailing slash because option disable",
			args: args{
				cfg: &config.Config{
					Server:      srvCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets: []*config.TargetConfig{
						{
							Name:   "target1",
							Bucket: bucketCfg,
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{
									Enabled:                                  true,
									RedirectWithTrailingSlashForNotFoundFile: false,
								},
							},
						},
					},
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/mount/not-found",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 404,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "Force trailing slash because option enable and file not found",
			args: args{
				cfg: &config.Config{
					Server:      srvCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets: []*config.TargetConfig{
						{
							Name:   "target1",
							Bucket: bucketCfg,
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{
									Enabled:                                  true,
									RedirectWithTrailingSlashForNotFoundFile: true,
								},
							},
						},
					},
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/mount/not-found",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 302,
			expectedHeaders: map[string]string{
				"Location": "/mount/not-found/",
			},
		},
		{
			name: "Don't force trailing slash because option enable and file found",
			args: args{
				cfg: &config.Config{
					Server:      srvCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets: []*config.TargetConfig{
						{
							Name:   "target1",
							Bucket: bucketCfg,
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{
									Enabled:                                  true,
									RedirectWithTrailingSlashForNotFoundFile: true,
								},
							},
						},
					},
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/mount/folder1/test.txt",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 200,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/plain; charset=utf-8",
			},
		},
		{
			name: "Don't force trailing slash because option enable and folder found",
			args: args{
				cfg: &config.Config{
					Server:      srvCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets: []*config.TargetConfig{
						{
							Name:   "target1",
							Bucket: bucketCfg,
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{
									Enabled:                                  true,
									RedirectWithTrailingSlashForNotFoundFile: true,
								},
							},
						},
					},
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/mount/folder1/",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 200,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
		{
			name: "Don't force trailing slash because option enable and folder not found",
			args: args{
				cfg: &config.Config{
					Server:      srvCfg,
					ListTargets: &config.ListTargetsConfig{},
					Tracing:     &config.TracingConfig{},
					Templates:   tplConfig,
					Targets: []*config.TargetConfig{
						{
							Name:   "target1",
							Bucket: bucketCfg,
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{
									Enabled:                                  true,
									RedirectWithTrailingSlashForNotFoundFile: true,
								},
							},
						},
					},
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/mount/not-found/",
			inputHeaders: map[string]string{
				"Origin": "https://test.fake",
				"Host":   "localhost",
			},
			expectedCode: 200,
			expectedHeaders: map[string]string{
				"Cache-Control": "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
				"Content-Type":  "text/html; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(tt.args.cfg)

			logger := log.NewLogger()
			// Create tracing service
			tsvc, err := tracing.New(cfgManagerMock, logger)
			assert.NoError(t, err)

			svr := &Server{
				logger:     logger,
				cfgManager: cfgManagerMock,
				metricsCl:  metricsCtx,
				tracingSvc: tsvc,
			}
			got, err := svr.generateRouter()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// If want error at this moment => stop
			if tt.wantErr {
				return
			}
			w := httptest.NewRecorder()
			req, err := http.NewRequest(
				tt.inputMethod,
				tt.inputURL,
				nil,
			)
			if err != nil {
				t.Error(err)
				return
			}

			// Set input headers
			if tt.inputHeaders != nil {
				for k, v := range tt.inputHeaders {
					req.Header.Set(k, v)
				}
			}

			got.ServeHTTP(w, req)

			if tt.expectedBody != "" {
				body := w.Body.String()
				if tt.expectedBody != body {
					t.Errorf("Integration test on TestTrailingSlashRedirect() body = \"%v\", expected body \"%v\"", body, tt.expectedBody)
				}
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				if tt.notExpectedBody == body {
					t.Errorf("Integration test on TestTrailingSlashRedirect() body = \"%v\", not expected body \"%v\"", body, tt.notExpectedBody)
				}
			}

			if tt.expectedHeaders != nil {
				for key, val := range tt.expectedHeaders {
					wheader := w.HeaderMap.Get(key)
					if val != wheader {
						t.Errorf("Integration test on TestTrailingSlashRedirect() header %s = %v, expected %v", key, wheader, val)
					}
				}
			}

			if tt.expectedCode != w.Code {
				t.Errorf("Integration test on TestTrailingSlashRedirect() status code = %v, expected status code %v", w.Code, tt.expectedCode)
			}
		})
	}
}
