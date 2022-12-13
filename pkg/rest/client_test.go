//go:build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rest

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockApi = api.NewApi("mock-api", "/mock-api", "", true, true, "", false)
var mockApiNotSingle = api.NewApi("mock-api", "/mock-api", "", false, true, "", false)

func TestNewClientNoUrl(t *testing.T) {
	_, err := NewDynatraceClient("", "abc")
	assert.ErrorContains(t, err, "no environment url")
}

func TestUrlSuffixGetsTrimmed(t *testing.T) {
	client, err := newDynatraceClient("https://my-environment.live.dynatrace.com/", "abc", nil, defaultRetrySettings)
	assert.NilError(t, err)
	assert.Equal(t, client.environmentUrl, "https://my-environment.live.dynatrace.com")
}

func TestNewClientNoToken(t *testing.T) {
	_, err := NewDynatraceClient("http://my-environment.live.dynatrace.com/", "")
	assert.ErrorContains(t, err, "no token")
}

func TestNewClientNoValidUrlLocalPath(t *testing.T) {
	_, err := NewDynatraceClient("/my-environment/live/dynatrace.com/", "abc")
	assert.ErrorContains(t, err, "not valid")
}

func TestNewClientNoValidUrlTypo(t *testing.T) {
	_, err := NewDynatraceClient("https//my-environment.live.dynatrace.com/", "abc")
	assert.ErrorContains(t, err, "not valid")
}

func TestNewClientNoValidUrlNoHttps(t *testing.T) {
	_, err := NewDynatraceClient("http//my-environment.live.dynatrace.com/", "abc")
	assert.ErrorContains(t, err, "not valid")
}

func TestNewClient(t *testing.T) {
	_, err := NewDynatraceClient("https://my-environment.live.dynatrace.com/", "abc")
	assert.NilError(t, err, "not valid")
}

func TestReadByIdReturnsAnErrorUponEncounteringAnError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "", http.StatusForbidden)
	}))
	defer func() { testServer.Close() }()
	client := newDynatraceClientForTesting(testServer)

	_, err := client.ReadById(mockApi, "test")
	assert.ErrorContains(t, err, "Response was")
}

func TestReadByIdEscapesTheId(t *testing.T) {
	unescapedId := "ruxit.perfmon.dotnetV4:%TimeInGC:time_in_gc_alert_high_generic"

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {}))
	defer func() { testServer.Close() }()
	client := newDynatraceClientForTesting(testServer)

	_, err := client.ReadById(mockApiNotSingle, unescapedId)
	assert.NilError(t, err)
}

func TestReadByIdReturnsTheResponseGivenNoError(t *testing.T) {
	body := []byte{1, 3, 3, 7}

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write(body)
	}))
	defer func() { testServer.Close() }()

	client := newDynatraceClientForTesting(testServer)

	resp, err := client.ReadById(mockApi, "test")
	assert.NilError(t, err, "there should not be an error")
	assert.DeepEqual(t, body, resp)
}

func TestListKnownSettings(t *testing.T) {

	tests := []struct {
		name                      string
		givenSchemaId             string
		givenServerResponses      []testServerResponse
		want                      KnownSettings
		wantQueryParamsPerApiCall [][]testQueryParams
		wantNumberOfApiCalls      int
		wantError                 bool
	}{
		{
			name:          "Lists Settings objects as expected",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
			},
			want: KnownSettings{
				"RG9jdG9yIFdobwo=": "f5823eca-4838-49d0-81d9-0514dd2c4640",
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", "externalId,objectId"},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:          "Handles Pagination when listing settings objects",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{200, `{ "items": [ {"objectId": "b1d4c623-25e0-4b54-9eb5-6734f1a72041", "externalId": "VGhlIE1hc3Rlcgo="} ] }`},
			},
			want: KnownSettings{
				"RG9jdG9yIFdobwo=": "f5823eca-4838-49d0-81d9-0514dd2c4640",
				"VGhlIE1hc3Rlcgo=": "b1d4c623-25e0-4b54-9eb5-6734f1a72041",
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", "externalId,objectId"},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfApiCalls: 2,
			wantError:            false,
		},
		{
			name:          "Returns empty if list if no items exist",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ ] }`},
			},
			want: KnownSettings{},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", "externalId,objectId"},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{400, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", "externalId,objectId"},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            true,
		},
		{
			name:          "Retries on HTTP error on paginated request and returns eventual success",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{400, `get next page fail`},
				{400, `retry fail`},
				{200, `{ "items": [ {"objectId": "b1d4c623-25e0-4b54-9eb5-6734f1a72041", "externalId": "VGhlIE1hc3Rlcgo="} ] }`},
			},
			want: KnownSettings{
				"RG9jdG9yIFdobwo=": "f5823eca-4838-49d0-81d9-0514dd2c4640",
				"VGhlIE1hc3Rlcgo=": "b1d4c623-25e0-4b54-9eb5-6734f1a72041",
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", "externalId,objectId"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfApiCalls: 4,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered getting further paginated responses",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{400, `get next page fail`},
				{400, `retry fail 1`},
				{400, `retry fail 2`},
				{400, `retry fail 3`},
			},
			want: nil,
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", "externalId,objectId"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfApiCalls: 5,
			wantError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.wantQueryParamsPerApiCall) > 0 {
					params := tt.wantQueryParamsPerApiCall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.Assert(t, addedQueryParameter != nil)
						assert.Assert(t, len(addedQueryParameter) > 0)
						assert.Equal(t, addedQueryParameter[0], param.value)
					}
				} else {
					assert.Equal(t, "", req.URL.RawQuery, "expected no query params - but '%s' was sent", req.URL.RawQuery)
				}

				resp := tt.givenServerResponses[apiCalls]
				if resp.statusCode != 200 {
					http.Error(rw, resp.body, resp.statusCode)
				} else {
					_, _ = rw.Write([]byte(resp.body))
				}

				apiCalls++
				assert.Check(t, apiCalls <= tt.wantNumberOfApiCalls, "expected at most %d API calls to happen, but encountered call %d", tt.wantNumberOfApiCalls, apiCalls)
			}))
			defer server.Close()

			client, err := newDynatraceClient(server.URL, "token", server.Client(), testRetrySettings)
			assert.NilError(t, err)

			res, err := client.ListKnownSettings(tt.givenSchemaId)

			if tt.wantError {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
			}

			assert.DeepEqual(t, res, tt.want)

			assert.Equal(t, apiCalls, tt.wantNumberOfApiCalls, "expected exactly %d API calls to happen but %d calls where made", tt.wantNumberOfApiCalls, apiCalls)
		})

	}
}
