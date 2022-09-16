// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
)

const (
	textPlain       = "text/plain; charset=utf-8"
	applicationJson = "application/json"
	contentType     = "Content-Type"
	acAllowOrigin   = "Access-Control-Allow-Origin"
)

func TestRest_Get(t *testing.T) {
	testCases := []struct {
		radix    int
		start, n int64
		want     string
	}{
		{10, 0, 0, ""},
		{10, 1, 0, ""},
		{10, 0, 1, "3"},
		{10, 1, 1, "1"},
		{10, 0, 50, "31415926535897932384626433832795028841971693993751"},
		{10, 1, 50, "14159265358979323846264338327950288419716939937510"},
		{10, 50_000_000_000_000 - 1, 2, "68"},
		{10, 50_000_000_000_000, 1, "8"},
		{16, 0, 0, ""},
		{16, 0, 1, "3"},
		{16, 1, 1, "2"},
		{16, 0, 50, "3243f6a8885a308d313198a2e03707344a4093822299f31d00"},
		{16, 1, 50, "243f6a8885a308d313198a2e03707344a4093822299f31d008"},
		{16, 41_524_101_186_051, 100, "717a7a8ddd2bac9e3f80609daacc0580794ca7ec01574c4c8209871b599d3548d16e177cb52cbbbe26f621b522b3e6bf1845"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("Radix %d Start %d N %d", tc.radix, tc.start, tc.n), func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/Get", nil)
			q := request.URL.Query()
			q.Add("start", strconv.FormatInt(tc.start, 10))
			q.Add("numberOfDigits", strconv.FormatInt(tc.n, 10))
			if tc.radix == 16 {
				q.Add("radix", strconv.Itoa(tc.radix))
			}
			request.URL.RawQuery = q.Encode()

			responseRecorder := httptest.NewRecorder()
			Get(responseRecorder, request)

			res := responseRecorder.Result()
			if res.StatusCode != http.StatusOK {
				t.Errorf("Get(): StatusCode, want = %d, got = %d", http.StatusOK, res.StatusCode)
			}
			if got := res.Header.Get(contentType); got != applicationJson {
				t.Errorf("Get(): %s, want = %s, got = %s", contentType, applicationJson, got)
			}
			if got := res.Header.Get(acAllowOrigin); got != "*" {
				t.Errorf("Get(): %s, want = *, got = %s", acAllowOrigin, got)
			}
			got := &GetResponse{}
			want := &GetResponse{Content: tc.want}
			if err := json.NewDecoder(res.Body).Decode(got); err != nil {
				t.Errorf("Get(): invalid json, error = %v", err)
			} else if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Get() = (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGet_BadRequests(t *testing.T) {
	testCases := []struct {
		radix    string
		start, n string
		message  string
	}{
		{"42", "0", "", "radix"},
		{"", "-1", "", "negative"},
		{"abc", "", "", "invalid"},
		{"", "9999999999999999999999", "", "invalid"},
		{"", "9223372036854775807", "", "out of range"},
		{"", "123", "-1", "negative"},
		{"16", "456", "~&!)#!", "invalid"},
		{"16", "", "1001", "too big"},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("Radix %v Start %v N %v",
			tc.radix, tc.start, tc.n), func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/Get", nil)
			q := request.URL.Query()
			q.Add("start", tc.start)
			q.Add("numberOfDigits", tc.n)
			q.Add("radix", tc.radix)
			request.URL.RawQuery = q.Encode()

			responseRecorder := httptest.NewRecorder()
			Get(responseRecorder, request)

			res := responseRecorder.Result()
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("Get(): StatusCode, want = %d, got = %d", http.StatusBadRequest, res.StatusCode)
			}
			if got := res.Header.Get(contentType); got != textPlain {
				t.Errorf("Get(): %s, want = %s, got = %s", contentType, textPlain, got)
			}
			if got := res.Header.Get(acAllowOrigin); got != "*" {
				t.Errorf("Get(): %s, want = *, got = %s", acAllowOrigin, got)
			}
			got, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("Failed to read response body: %v", err)
			}
			if !strings.Contains(string(got), tc.message) {
				t.Errorf("Get() should contain %s, got:\n%s", tc.message, got)
			}
			if strings.Contains(string(got), "\"content\"") {
				t.Errorf("Get() should not contain \"content\", got:\n%s", got)
			}
		})
	}
}

func TestRest_NotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/NotFound", nil)
	responseRecorder := httptest.NewRecorder()
	NotFound(responseRecorder, request)
	res := responseRecorder.Result()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("NotFound(): StatusCode, want = %d, got = %d", http.StatusNotFound, res.StatusCode)
	}
	if got := res.Header.Get(contentType); got != textPlain {
		t.Errorf("NotFound(): %s, want = %s, got = %s", contentType, textPlain, got)
	}
	got, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Failed to read response body: %v", err)
	}
	const want = "The requested url /NotFound was not found.\n"
	if diff := cmp.Diff(want, string(got)); diff != "" {
		t.Errorf("NotFound(): (-want, +got):\n%s", diff)
	}
}
